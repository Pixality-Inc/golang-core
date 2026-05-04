package sftp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path"
	"sync"
	"time"

	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/storage"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const defaultDialTimeout = 30 * time.Second

var (
	ErrNoChunks        = errors.New("sftp: compose called with no chunks")
	ErrEmptyDeletePath = errors.New("sftp: refusing DeleteDir with empty path")
)

type Client interface {
	Close()
	Upload(ctx context.Context, objectName string, file io.Reader) error
	UploadFile(ctx context.Context, objectName string, filename string) error
	DeleteDir(ctx context.Context, objectName string) error
	Delete(ctx context.Context, objectName string) error
	Download(ctx context.Context, objectName string) (io.ReadCloser, error)
	DownloadFile(ctx context.Context, objectName string, filename string) error
	ReadDir(ctx context.Context, objectName string) ([]storage.DirEntry, error)
	MkDir(ctx context.Context, objectName string) error
	FileExists(ctx context.Context, objectName string) (bool, error)
	Compose(ctx context.Context, objectName string, chunks []string) error
}

type Impl struct {
	log                  logger.Loggable
	name                 string
	host                 string
	port                 int
	username             string
	password             string
	privateKey           []byte
	privateKeyPassphrase []byte
	baseDir              string
	sshClient            *ssh.Client
	sftpClient           *sftp.Client
	mutex                sync.Mutex
}

func NewClient(
	name string,
	host string,
	port int,
	username string,
	password string,
	privateKey []byte,
	privateKeyPassphrase []byte,
	baseDir string,
) Client {
	return &Impl{
		log: logger.NewLoggableImplWithServiceAndFields(
			"sftp",
			logger.Fields{
				"name": name,
				"host": host,
			},
		),
		name:                 name,
		host:                 host,
		port:                 port,
		username:             username,
		password:             password,
		privateKey:           privateKey,
		privateKeyPassphrase: privateKeyPassphrase,
		baseDir:              baseDir,
	}
}

func (c *Impl) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.sftpClient != nil {
		if err := c.sftpClient.Close(); err != nil {
			c.log.GetLoggerWithoutContext().WithError(err).Error("sftp close failed")
		}
	}

	if c.sshClient != nil {
		if err := c.sshClient.Close(); err != nil {
			c.log.GetLoggerWithoutContext().WithError(err).Error("ssh close failed")
		}
	}

	c.sftpClient = nil
	c.sshClient = nil
}

func (c *Impl) Upload(ctx context.Context, objectName string, file io.Reader) error {
	log := c.log.GetLogger(ctx)
	log.Infof("Uploading object '%s'", objectName)

	if err := c.init(ctx); err != nil {
		return err
	}

	objectFullName := c.getObjectFullName(objectName)
	if err := c.sftpClient.MkdirAll(path.Dir(objectFullName)); err != nil {
		return fmt.Errorf("sftp: mkdir for '%s': %w", objectFullName, err)
	}

	dest, err := c.sftpClient.Create(objectFullName)
	if err != nil {
		return fmt.Errorf("sftp: create '%s': %w", objectFullName, err)
	}

	defer func() {
		if closeErr := dest.Close(); closeErr != nil {
			log.WithError(closeErr).Errorf("failed to close writer for '%s'", objectFullName)
		}
	}()

	if _, err = io.Copy(dest, file); err != nil {
		return fmt.Errorf("sftp: upload '%s': %w", objectFullName, err)
	}

	return nil
}

func (c *Impl) UploadFile(ctx context.Context, objectName string, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			c.log.GetLogger(ctx).WithError(closeErr).Errorf("failed to close file '%s'", filename)
		}
	}()

	return c.Upload(ctx, objectName, file)
}

func (c *Impl) Delete(ctx context.Context, objectName string) error {
	c.log.GetLogger(ctx).Infof("Deleting object '%s'", objectName)

	if err := c.init(ctx); err != nil {
		return err
	}

	objectFullName := c.getObjectFullName(objectName)
	if err := c.sftpClient.Remove(objectFullName); err != nil {
		return fmt.Errorf("sftp: delete '%s': %w", objectFullName, err)
	}

	return nil
}

func (c *Impl) DeleteDir(ctx context.Context, objectName string) error {
	c.log.GetLogger(ctx).Infof("Deleting directory '%s'", objectName)

	if c.baseDir == "" && objectName == "" {
		return ErrEmptyDeletePath
	}

	if err := c.init(ctx); err != nil {
		return err
	}

	objectFullName := c.getObjectFullName(objectName)
	if err := c.removeAll(objectFullName); err != nil {
		return fmt.Errorf("sftp: delete dir '%s': %w", objectFullName, err)
	}

	return nil
}

func (c *Impl) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	c.log.GetLogger(ctx).Infof("Downloading object '%s'", objectName)

	if err := c.init(ctx); err != nil {
		return nil, err
	}

	objectFullName := c.getObjectFullName(objectName)
	file, err := c.sftpClient.Open(objectFullName)
	if err != nil {
		return nil, fmt.Errorf("sftp: download '%s': %w", objectFullName, err)
	}

	return file, nil
}

func (c *Impl) DownloadFile(ctx context.Context, objectName string, filename string) error {
	readCloser, err := c.Download(ctx, objectName)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := readCloser.Close(); closeErr != nil {
			c.log.GetLogger(ctx).WithError(closeErr).Errorf("failed to close reader '%s'", objectName)
		}
	}()

	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer func() {
		if closeErr := outFile.Close(); closeErr != nil {
			c.log.GetLogger(ctx).WithError(closeErr).Errorf("failed to close file '%s'", filename)
		}
	}()

	if _, err := io.Copy(outFile, readCloser); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	return nil
}

func (c *Impl) ReadDir(ctx context.Context, objectName string) ([]storage.DirEntry, error) {
	if err := c.init(ctx); err != nil {
		return nil, err
	}

	objectFullName := c.getObjectFullName(objectName)
	infos, err := c.sftpClient.ReadDir(objectFullName)
	if err != nil {
		return nil, fmt.Errorf("sftp: read dir '%s': %w", objectFullName, err)
	}

	entries := make([]storage.DirEntry, len(infos))
	for i, info := range infos {
		entries[i] = fileInfoDirEntry{info: info}
	}

	return entries, nil
}

func (c *Impl) MkDir(ctx context.Context, objectName string) error {
	if err := c.init(ctx); err != nil {
		return err
	}

	objectFullName := c.getObjectFullName(objectName)
	if err := c.sftpClient.MkdirAll(objectFullName); err != nil {
		return fmt.Errorf("sftp: mkdir '%s': %w", objectFullName, err)
	}

	return nil
}

func (c *Impl) FileExists(ctx context.Context, objectName string) (bool, error) {
	if err := c.init(ctx); err != nil {
		return false, err
	}

	objectFullName := c.getObjectFullName(objectName)
	_, err := c.sftpClient.Stat(objectFullName)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("sftp: stat '%s': %w", objectFullName, err)
	}

	return true, nil
}

func (c *Impl) Compose(ctx context.Context, objectName string, chunks []string) error {
	log := c.log.GetLogger(ctx)
	log.Infof("Composing object '%s' from %d chunks", objectName, len(chunks))

	if len(chunks) == 0 {
		return fmt.Errorf("%w: '%s'", ErrNoChunks, objectName)
	}

	if err := c.init(ctx); err != nil {
		return err
	}

	objectFullName := c.getObjectFullName(objectName)
	if err := c.sftpClient.MkdirAll(path.Dir(objectFullName)); err != nil {
		return fmt.Errorf("sftp: mkdir for '%s': %w", objectFullName, err)
	}

	dest, err := c.sftpClient.Create(objectFullName)
	if err != nil {
		return fmt.Errorf("sftp: create composed '%s': %w", objectFullName, err)
	}

	defer func() {
		if closeErr := dest.Close(); closeErr != nil {
			log.WithError(closeErr).Errorf("failed to close composed file '%s'", objectFullName)
		}
	}()

	for _, chunk := range chunks {
		if err := c.copyChunk(ctx, dest, chunk); err != nil {
			return err
		}
	}

	return nil
}

func (c *Impl) init(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.sftpClient != nil {
		return nil
	}

	auth, err := c.authMethods()
	if err != nil {
		return err
	}

	config := &ssh.ClientConfig{
		User:            c.username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // compatibility with existing simple client constructors.
		Timeout:         defaultDialTimeout,
	}

	port := c.port
	if port == 0 {
		port = 22
	}

	dialer := net.Dialer{Timeout: defaultDialTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", c.host, port))
	if err != nil {
		return fmt.Errorf("sftp: dial %s:%d: %w", c.host, port, err)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, fmt.Sprintf("%s:%d", c.host, port), config)
	if err != nil {
		_ = conn.Close()

		return fmt.Errorf("sftp: ssh connect %s:%d: %w", c.host, port, err)
	}

	c.sshClient = ssh.NewClient(sshConn, chans, reqs)

	sftpClient, err := sftp.NewClient(c.sshClient)
	if err != nil {
		_ = c.sshClient.Close()
		c.sshClient = nil

		return fmt.Errorf("sftp: create client: %w", err)
	}

	c.sftpClient = sftpClient

	return nil
}

func (c *Impl) authMethods() ([]ssh.AuthMethod, error) {
	auth := make([]ssh.AuthMethod, 0, 2)
	if c.password != "" {
		auth = append(auth, ssh.Password(c.password))
	}

	if len(c.privateKey) > 0 {
		signer, err := parsePrivateKey(c.privateKey, c.privateKeyPassphrase)
		if err != nil {
			return nil, err
		}

		auth = append(auth, ssh.PublicKeys(signer))
	}

	if len(auth) == 0 {
		return nil, errors.New("sftp: no auth method provided")
	}

	return auth, nil
}

func parsePrivateKey(privateKey, passphrase []byte) (ssh.Signer, error) {
	if len(passphrase) > 0 {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(privateKey, passphrase)
		if err != nil {
			return nil, fmt.Errorf("sftp: parse encrypted private key: %w", err)
		}

		return signer, nil
	}

	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("sftp: parse private key: %w", err)
	}

	return signer, nil
}

func (c *Impl) removeAll(remotePath string) error {
	info, err := c.sftpClient.Stat(remotePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return c.sftpClient.Remove(remotePath)
	}

	entries, err := c.sftpClient.ReadDir(remotePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if err := c.removeAll(path.Join(remotePath, entry.Name())); err != nil {
			return err
		}
	}

	return c.sftpClient.RemoveDirectory(remotePath)
}

func (c *Impl) copyChunk(ctx context.Context, dest io.Writer, chunk string) error {
	chunkFullName := c.getObjectFullName(chunk)
	source, err := c.sftpClient.Open(chunkFullName)
	if err != nil {
		return fmt.Errorf("sftp: open chunk '%s': %w", chunkFullName, err)
	}

	defer func() {
		if closeErr := source.Close(); closeErr != nil {
			c.log.GetLogger(ctx).WithError(closeErr).Errorf("failed to close chunk '%s'", chunkFullName)
		}
	}()

	if _, err = io.Copy(dest, source); err != nil {
		return fmt.Errorf("sftp: copy chunk '%s': %w", chunkFullName, err)
	}

	return nil
}

func (c *Impl) getObjectFullName(objectName string) string {
	if c.baseDir == "" && objectName == "" {
		return "."
	}
	if c.baseDir == "" {
		return path.Clean(objectName)
	}
	if objectName == "" {
		return path.Clean(c.baseDir)
	}

	return path.Join(c.baseDir, objectName)
}

type fileInfoDirEntry struct {
	info fs.FileInfo
}

func (d fileInfoDirEntry) IsDir() bool {
	return d.info.IsDir()
}

func (d fileInfoDirEntry) Name() string {
	return d.info.Name()
}

func (d fileInfoDirEntry) Type() fs.FileMode {
	return d.info.Mode().Type()
}

func (d fileInfoDirEntry) Info() (fs.FileInfo, error) {
	return d.info, nil
}
