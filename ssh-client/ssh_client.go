package ssh_client

import (
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"
	"time"
)

const USER = "www"
const PWD = "yuxingfei"
const HOST = "10.1.2.179"
const PORT = 22

//连接
func Connect() (*sftp.Client, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		sshClient    *ssh.Client
		sftpClient   *sftp.Client
		err          error
	)
	// get auth method
	auth = make([]ssh.AuthMethod, 0)
	auth = append(auth, ssh.Password(PWD))

	clientConfig = &ssh.ClientConfig{
		User:    USER,
		Auth:    auth,
		Timeout: 30 * time.Second,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	// connet to ssh
	addr = fmt.Sprintf("%s:%d", HOST, PORT)

	if sshClient, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}

	// create sftp client
	if sftpClient, err = sftp.NewClient(sshClient); err != nil {
		return nil, err
	}

	return sftpClient, nil
}

//上传文件
func uploadFile(sftpClient *sftp.Client, localFilePath string, remotePath string) string {
	//打开本地文件流
	srcFile, err := os.Open(localFilePath)
	if err != nil {
		return "os.Open error : " + err.Error() + " " + localFilePath
	}
	//关闭文件流
	defer srcFile.Close()
	//上传到远端服务器的文件名,与本地路径末尾相同
	//var remoteFileName = path.Base(localFilePath)
	//打开远程文件,如果不存在就创建一个
	dstFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return "sftpClient.Create error : " + err.Error() + " " + remotePath
	}
	//关闭远程文件
	defer dstFile.Close()
	//读取本地文件,写入到远程文件中(这里没有分快穿,自己写的话可以改一下,防止内存溢出)
	ff, err := ioutil.ReadAll(srcFile)
	if err != nil {
		return "ReadAll error : " + err.Error() + " " + localFilePath
	}
	dstFile.Write(ff)
	return localFilePath + "  copy file to remote server finished!"
}

//上传目录
func uploadDirectory(sftpClient *sftp.Client, localPath string, remotePath string) string {
	//打开本地文件夹流
	localFiles, err := ioutil.ReadDir(localPath)
	if err != nil {
		return "路径错误 error :" + err.Error()
	}
	//先创建最外层文件夹
	sftpClient.Mkdir(remotePath)
	//遍历文件夹内容
	for _, backupDir := range localFiles {
		localFilePath := path.Join(localPath, backupDir.Name())
		remoteFilePath := path.Join(remotePath, backupDir.Name())
		//判断是否是文件,是文件直接上传.是文件夹,先远程创建文件夹,再递归复制内部文件
		if backupDir.IsDir() {
			sftpClient.Mkdir(remoteFilePath)
			uploadDirectory(sftpClient, localFilePath, remoteFilePath)
		} else {
			remoteAllPath := strings.TrimRight(remotePath, "/") + "/" + backupDir.Name()
			uploadFile(sftpClient, path.Join(localPath, backupDir.Name()), remoteAllPath)
		}
	}

	return localPath + "  copy directory to remote server finished!"
}

func Upload(sftpClient *sftp.Client, localPath string, remotePath string) string {
	//获取路径的属性
	s, err := os.Stat(localPath)
	if err != nil {
		return "文件路径不存在"
	}
	//判断是否是文件夹
	var msg string
	if s.IsDir() {
		msg = uploadDirectory(sftpClient, localPath, remotePath)
	} else {
		msg = uploadFile(sftpClient, localPath, remotePath)
	}
	return msg
}
