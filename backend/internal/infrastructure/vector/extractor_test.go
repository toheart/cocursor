package vector

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArchiveExtractor_ExtractTarGz(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 创建测试 tar.gz 文件
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")
	createTestTarGz(t, tarGzPath, map[string]string{
		"file1.txt":     "content of file 1",
		"dir/file2.txt": "content of file 2",
	})

	// 创建解压器
	extractor := NewArchiveExtractor()

	// 解压
	extractDir := filepath.Join(tmpDir, "extracted")
	err := extractor.Extract(tarGzPath, extractDir)
	require.NoError(t, err)

	// 验证文件存在
	content1, err := os.ReadFile(filepath.Join(extractDir, "file1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content of file 1", string(content1))

	content2, err := os.ReadFile(filepath.Join(extractDir, "dir", "file2.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content of file 2", string(content2))
}

func TestArchiveExtractor_ExtractZip(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 创建测试 zip 文件
	zipPath := filepath.Join(tmpDir, "test.zip")
	createTestZip(t, zipPath, map[string]string{
		"file1.txt":     "zip content 1",
		"dir/file2.txt": "zip content 2",
	})

	// 创建解压器
	extractor := NewArchiveExtractor()

	// 解压
	extractDir := filepath.Join(tmpDir, "extracted")
	err := extractor.Extract(zipPath, extractDir)
	require.NoError(t, err)

	// 验证文件存在
	content1, err := os.ReadFile(filepath.Join(extractDir, "file1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "zip content 1", string(content1))

	content2, err := os.ReadFile(filepath.Join(extractDir, "dir", "file2.txt"))
	require.NoError(t, err)
	assert.Equal(t, "zip content 2", string(content2))
}

func TestArchiveExtractor_PathTraversalProtection(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 创建包含路径遍历攻击的 tar.gz 文件
	tarGzPath := filepath.Join(tmpDir, "malicious.tar.gz")
	createMaliciousTarGz(t, tarGzPath)

	// 创建解压器
	extractor := NewArchiveExtractor()

	// 解压应该失败
	extractDir := filepath.Join(tmpDir, "extracted")
	err := extractor.Extract(tarGzPath, extractDir)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrPathTraversal)
}

func TestArchiveExtractor_UnsupportedFormat(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 创建一个假文件
	fakePath := filepath.Join(tmpDir, "test.rar")
	err := os.WriteFile(fakePath, []byte("fake rar content"), 0644)
	require.NoError(t, err)

	// 创建解压器
	extractor := NewArchiveExtractor()

	// 解压应该失败
	extractDir := filepath.Join(tmpDir, "extracted")
	err = extractor.Extract(fakePath, extractDir)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedArchive)
}

func TestArchiveExtractor_FindBinary(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 创建目录结构
	err := os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	require.NoError(t, err)

	// 创建二进制文件
	binaryPath := filepath.Join(tmpDir, "subdir", "qdrant")
	err = os.WriteFile(binaryPath, []byte("binary content"), 0755)
	require.NoError(t, err)

	// 创建解压器
	extractor := NewArchiveExtractor()

	// 查找二进制文件
	foundPath, err := extractor.FindBinary(tmpDir, "qdrant")
	require.NoError(t, err)
	assert.Equal(t, binaryPath, foundPath)
}

func TestArchiveExtractor_FindBinary_NotFound(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 创建解压器
	extractor := NewArchiveExtractor()

	// 查找不存在的二进制文件
	_, err := extractor.FindBinary(tmpDir, "nonexistent")
	assert.Error(t, err)
}

// createTestTarGz 创建测试用的 tar.gz 文件
func createTestTarGz(t *testing.T, path string, files map[string]string) {
	t.Helper()

	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	for name, content := range files {
		// 如果有目录，先创建目录条目
		dir := filepath.Dir(name)
		if dir != "." {
			err := tarWriter.WriteHeader(&tar.Header{
				Name:     dir + "/",
				Mode:     0755,
				Typeflag: tar.TypeDir,
			})
			require.NoError(t, err)
		}

		// 写入文件
		err := tarWriter.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		})
		require.NoError(t, err)

		_, err = tarWriter.Write([]byte(content))
		require.NoError(t, err)
	}
}

// createTestZip 创建测试用的 zip 文件
func createTestZip(t *testing.T, path string, files map[string]string) {
	t.Helper()

	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	for name, content := range files {
		writer, err := zipWriter.Create(name)
		require.NoError(t, err)

		_, err = writer.Write([]byte(content))
		require.NoError(t, err)
	}
}

// createMaliciousTarGz 创建包含路径遍历攻击的 tar.gz 文件
func createMaliciousTarGz(t *testing.T, path string) {
	t.Helper()

	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// 尝试写入到父目录
	maliciousContent := "malicious content"
	err = tarWriter.WriteHeader(&tar.Header{
		Name: "../../../etc/malicious",
		Mode: 0644,
		Size: int64(len(maliciousContent)),
	})
	require.NoError(t, err)

	_, err = tarWriter.Write([]byte(maliciousContent))
	require.NoError(t, err)
}
