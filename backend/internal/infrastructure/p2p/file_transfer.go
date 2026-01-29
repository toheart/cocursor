package p2p

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// FileTransfer 文件传输工具
type FileTransfer struct {
	httpClient *http.Client
}

// NewFileTransfer 创建文件传输工具
func NewFileTransfer() *FileTransfer {
	return &FileTransfer{
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // 下载超时 5 分钟
		},
	}
}

// PackDirectory 打包目录为 tar.gz
func (t *FileTransfer) PackDirectory(dirPath string) ([]byte, string, error) {
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzWriter)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		// 跳过根目录
		if relPath == "." {
			return nil
		}

		// 创建 tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath) // 使用正斜杠

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// 写入文件内容
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}

			if _, err := io.Copy(tarWriter, file); err != nil {
				_ = file.Close()
				return err
			}
			_ = file.Close()
		}

		return nil
	})

	if err != nil {
		return nil, "", err
	}

	if err := tarWriter.Close(); err != nil {
		return nil, "", err
	}
	if err := gzWriter.Close(); err != nil {
		return nil, "", err
	}

	data := buf.Bytes()
	checksum := t.CalculateChecksum(data)

	return data, checksum, nil
}

// UnpackArchive 解包 tar.gz 到目录
func (t *FileTransfer) UnpackArchive(data []byte, destDir string) error {
	gzReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// 安全检查：防止路径遍历攻击
		targetPath := filepath.Join(destDir, filepath.FromSlash(header.Name))
		if !isSubPath(destDir, targetPath) {
			return fmt.Errorf("invalid file path: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// 确保父目录存在
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			file, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				_ = file.Close()
				return fmt.Errorf("failed to write file: %w", err)
			}
			_ = file.Close()

			// 设置文件权限（忽略权限设置错误，Windows 上可能失败）
			_ = os.Chmod(targetPath, os.FileMode(header.Mode))
		}
	}

	return nil
}

// isSubPath 检查 path 是否是 basePath 的子路径
func isSubPath(basePath, path string) bool {
	rel, err := filepath.Rel(basePath, path)
	if err != nil {
		return false
	}
	return rel != ".." && !filepath.IsAbs(rel) && rel[:2] != ".."
}

// CalculateChecksum 计算数据的校验和
func (t *FileTransfer) CalculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// CalculateDirectoryChecksum 计算目录的校验和
func (t *FileTransfer) CalculateDirectoryChecksum(dirPath string) (string, error) {
	hash := sha256.New()

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// 添加相对路径到哈希
		relPath, _ := filepath.Rel(dirPath, path)
		hash.Write([]byte(relPath))

		// 添加文件内容到哈希
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.Copy(hash, file); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// VerifyChecksum 验证校验和
func (t *FileTransfer) VerifyChecksum(data []byte, expectedChecksum string) bool {
	return t.CalculateChecksum(data) == expectedChecksum
}

// DownloadFile 从 URL 下载文件
func (t *FileTransfer) DownloadFile(url string) ([]byte, error) {
	resp, err := t.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return data, nil
}

// DownloadSkill 下载技能文件
func (t *FileTransfer) DownloadSkill(endpoint, skillID string) ([]byte, error) {
	url := fmt.Sprintf("http://%s/p2p/skills/%s/download", endpoint, skillID)
	return t.DownloadFile(url)
}

// GetSkillMeta 获取技能元数据
func (t *FileTransfer) GetSkillMeta(endpoint, skillID string) (*SkillMeta, error) {
	url := fmt.Sprintf("http://%s/p2p/skills/%s/meta", endpoint, skillID)

	resp, err := t.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get meta: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get meta failed with status: %d", resp.StatusCode)
	}

	var meta SkillMeta
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := jsonUnmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// GetDirectoryInfo 获取目录信息
func (t *FileTransfer) GetDirectoryInfo(dirPath string) (files []string, totalSize int64, err error) {
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(dirPath, path)
		files = append(files, filepath.ToSlash(relPath))
		totalSize += info.Size()

		return nil
	})

	return
}

// jsonUnmarshal JSON 反序列化
func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
