package vector

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"log/slog"

	"github.com/cocursor/backend/internal/infrastructure/log"
)

// 解压相关错误
var (
	ErrUnsupportedArchive = errors.New("unsupported archive format")
	ErrPathTraversal      = errors.New("path traversal detected in archive")
	ErrExtractFailed      = errors.New("extraction failed")
)

// Extractor 归档解压器接口
type Extractor interface {
	// Extract 解压归档文件到指定目录
	// archivePath: 归档文件路径
	// destDir: 目标目录
	Extract(archivePath, destDir string) error

	// FindBinary 在解压目录中查找指定的二进制文件
	// extractDir: 解压后的目录
	// binaryName: 二进制文件名
	FindBinary(extractDir, binaryName string) (string, error)
}

// ArchiveExtractor 归档解压器实现
type ArchiveExtractor struct {
	logger *slog.Logger
}

// NewArchiveExtractor 创建新的归档解压器
func NewArchiveExtractor() *ArchiveExtractor {
	return &ArchiveExtractor{
		logger: log.NewModuleLogger("vector", "extractor"),
	}
}

// Extract 实现 Extractor 接口
func (e *ArchiveExtractor) Extract(archivePath, destDir string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}

	// 根据文件扩展名选择解压方式
	lowerPath := strings.ToLower(archivePath)
	if strings.HasSuffix(lowerPath, ".zip") {
		return e.extractZip(archivePath, destDir)
	} else if strings.HasSuffix(lowerPath, ".tar.gz") || strings.HasSuffix(lowerPath, ".tgz") {
		return e.extractTarGz(archivePath, destDir)
	}

	return fmt.Errorf("%w: %s", ErrUnsupportedArchive, filepath.Ext(archivePath))
}

// extractTarGz 解压 tar.gz 文件
func (e *ArchiveExtractor) extractTarGz(tarGzPath, destDir string) error {
	// 打开文件
	file, err := os.Open(tarGzPath)
	if err != nil {
		return fmt.Errorf("failed to open tar.gz file: %w", err)
	}
	defer file.Close()

	// 创建 gzip 读取器
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// 创建 tar 读取器
	tarReader := tar.NewReader(gzReader)

	// 遍历 tar 文件中的所有条目
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		// 构建目标路径并验证安全性
		targetPath, err := e.safeJoin(destDir, header.Name)
		if err != nil {
			return err
		}

		// 根据文件类型处理
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, e.getPermission(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			if err := e.extractFile(tarReader, targetPath, header.Mode); err != nil {
				return err
			}

		case tar.TypeSymlink:
			if err := e.createSymlink(header.Linkname, targetPath, destDir); err != nil {
				// 符号链接失败只记录警告，不阻止解压
				e.logger.Warn("failed to create symlink",
					"target", targetPath,
					"link", header.Linkname,
					"error", err)
			}

		default:
			// 忽略其他类型的条目
			e.logger.Debug("skipping unsupported tar entry type",
				"type", header.Typeflag,
				"name", header.Name)
		}
	}

	return nil
}

// extractZip 解压 ZIP 文件
func (e *ArchiveExtractor) extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		// 构建目标路径并验证安全性
		targetPath, err := e.safeJoin(destDir, f.Name)
		if err != nil {
			return err
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, e.getPermission(int64(f.Mode()))); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		// 确保父目录存在
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// 解压文件
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		if err := e.writeFile(rc, targetPath, int64(f.Mode())); err != nil {
			rc.Close()
			return err
		}
		rc.Close()
	}

	return nil
}

// extractFile 从 tar 读取器中提取文件
func (e *ArchiveExtractor) extractFile(reader io.Reader, targetPath string, mode int64) error {
	// 确保父目录存在
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for file: %w", err)
	}

	return e.writeFile(reader, targetPath, mode)
}

// writeFile 将内容写入文件
func (e *ArchiveExtractor) writeFile(reader io.Reader, targetPath string, mode int64) error {
	outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, e.getPermission(mode))
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, reader); err != nil {
		return fmt.Errorf("failed to extract file content: %w", err)
	}

	return nil
}

// safeJoin 安全地连接路径，防止路径遍历攻击
func (e *ArchiveExtractor) safeJoin(destDir, entryPath string) (string, error) {
	// 清理路径
	cleanDest := filepath.Clean(destDir)
	targetPath := filepath.Join(destDir, entryPath)
	cleanTarget := filepath.Clean(targetPath)

	// 验证目标路径是否在目标目录内
	if !strings.HasPrefix(cleanTarget, cleanDest+string(os.PathSeparator)) && cleanTarget != cleanDest {
		return "", fmt.Errorf("%w: %s", ErrPathTraversal, entryPath)
	}

	return cleanTarget, nil
}

// getPermission 获取适合当前平台的文件权限
func (e *ArchiveExtractor) getPermission(mode int64) os.FileMode {
	if runtime.GOOS == "windows" {
		// Windows 不支持 Unix 权限，使用默认权限
		return 0755
	}
	return os.FileMode(mode)
}

// createSymlink 创建符号链接（跨平台处理）
func (e *ArchiveExtractor) createSymlink(linkTarget, linkPath, destDir string) error {
	if runtime.GOOS == "windows" {
		// Windows 创建符号链接需要特殊权限
		// 改为尝试复制文件（如果目标存在）
		targetPath := filepath.Join(destDir, linkTarget)
		if info, err := os.Stat(targetPath); err == nil && !info.IsDir() {
			return copyFile(targetPath, linkPath)
		}
		// 如果目标不存在，记录警告但不失败
		e.logger.Warn("cannot create symlink on Windows, target not found",
			"target", linkTarget,
			"link", linkPath)
		return nil
	}

	// 非 Windows 系统正常创建符号链接
	return os.Symlink(linkTarget, linkPath)
}

// FindBinary 在解压目录中查找二进制文件
func (e *ArchiveExtractor) FindBinary(extractDir, binaryName string) (string, error) {
	var foundPath string

	err := filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 继续遍历
		}
		if !info.IsDir() && info.Name() == binaryName {
			foundPath = path
			return filepath.SkipAll // 找到后停止遍历
		}
		return nil
	})

	if err != nil && !errors.Is(err, filepath.SkipAll) {
		return "", fmt.Errorf("failed to search for binary: %w", err)
	}

	if foundPath == "" {
		return "", fmt.Errorf("binary %s not found in %s", binaryName, extractDir)
	}

	return foundPath, nil
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// 获取源文件信息
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// 设置文件权限（非 Windows）
	if runtime.GOOS != "windows" {
		if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
			return fmt.Errorf("failed to set file permissions: %w", err)
		}
	}

	return nil
}
