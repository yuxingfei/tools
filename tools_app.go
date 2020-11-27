// Copyright 2012 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
	"github.com/pkg/sftp"
	"gui/tools_app/ssh"
	ssh_client "gui/tools_app/ssh-client"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//手机号加解密
type Phone struct {
	PhoneEncrypt string
	PhoneDecrypt string
}

//md5加密
type Md5 struct {
	Md5Str string
}

//代码同步
type CodeRsyncItem struct {
	MapItems  map[int]string
	CheckItem int
}

type TempTextStruct struct {
	TempTextString string
}

//ssh客户端
var sshCli *ssh.Cli

//sFtp客户端
var sFtpCli *sftp.Client

//ssh目录初始化
var projectNameArr []string

func main() {
	var mw *walk.MainWindow
	var outTE *walk.TextEdit
	var err error
	phone := new(Phone)
	md5str := new(Md5)
	codeRsyncItem := new(CodeRsyncItem)
	codeRsyncItem.MapItems = make(map[int]string)

	tempTextStruct := new(TempTextStruct)

	//ssh客户端
	go func() {
		sshCli = ssh.New(ssh.IP, ssh.USER, ssh.PWD, ssh.PORT)
		//第一次初始化，查看目录文件夹
		if len(projectNameArr) == 0 {
			out, err := sshCli.Run("ls")
			if err != nil {
				walk.MsgBox(mw, "title", "ssh run error. err:"+err.Error(), walk.MsgBoxIconInformation)
			}
			projectNameArr = strings.Split(out, "\n")
		}
		sFtpCli, err = ssh_client.Connect()
		if err != nil {
			walk.MsgBox(mw, "Error", err.Error(), walk.MsgBoxIconError)
		}
	}()

	if err := (MainWindow{
		AssignTo: &mw,
		Title:    "便捷工具",
		Icon:     "favicon.ico",
		Size:     Size{460, 600},
		Layout:   VBox{},
		Background: GradientBrush{
			Vertexes: []walk.GradientVertex{
				{X: 0, Y: 0, Color: walk.RGB(255, 255, 127)},
				{X: 1, Y: 0, Color: walk.RGB(127, 191, 255)},
				{X: 0.5, Y: 0.5, Color: walk.RGB(255, 255, 255)},
				{X: 1, Y: 1, Color: walk.RGB(127, 255, 127)},
				{X: 0, Y: 1, Color: walk.RGB(255, 127, 127)},
			},
			Triangles: []walk.GradientTriangle{
				{0, 1, 2},
				{1, 3, 2},
				{3, 4, 2},
				{4, 0, 2},
			},
		},
		Children: []Widget{
			PushButton{
				Text:    "代码同步",
				MinSize: Size{60, 60},
				OnClicked: func() {
					//本机授权
					ip := getLocalIpAddr()
					if ip != "10.1.2.49" {
						walk.MsgBox(mw, "title", "本机没有授权", walk.MsgBoxIconInformation)
						return
					}

					if cmd, sshCli, err := RunCodeRsyncDialog(sshCli, mw, codeRsyncItem); err != nil {
						outTE.SetText(err.Error())
					} else if cmd == walk.DlgCmdOK {
						projectName := codeRsyncItem.MapItems[codeRsyncItem.CheckItem]
						shellStr := "cd " + projectName + " && git pull origin master"
						output, err := sshCli.Run(shellStr)
						if err != nil {
							outTE.SetText("err:" + err.Error())
						} else {
							outTE.SetText(strings.ReplaceAll(fmt.Sprintf("%v", output), "\n", "\r\n"))
						}
					}
				},
			},
			PushButton{
				Text:    "MD5加密",
				MinSize: Size{60, 60},
				OnClicked: func() {
					if cmd, err := RunMd5EncryptDialog(mw, md5str); err != nil {
						outTE.SetText(err.Error())
					} else if cmd == walk.DlgCmdOK {
						if md5str.Md5Str == "" {
							outTE.SetText("没有输入需要MD5加密的字符串")
							return
						}
						msg := Md5StringEncrypt(md5str.Md5Str)
						outTE.SetText(msg)
					}
				},
			},
			PushButton{
				Text:    "手机号加密",
				MinSize: Size{60, 60},
				OnClicked: func() {
					if cmd, err := RunPhoneEncryptDialog(mw, phone); err != nil {
						outTE.SetText(err.Error())
					} else if cmd == walk.DlgCmdOK {
						if phone.PhoneDecrypt == "" {
							outTE.SetText("没有输入手机号码")
							return
						}
						//手机号码解密
						msg := phoneEncrypt(phone.PhoneDecrypt)
						outTE.SetText(msg)
					}
				},
			},
			PushButton{
				Text:    "手机号解密",
				MinSize: Size{60, 60},
				OnClicked: func() {
					if cmd, err := RunPhoneDecryptDialog(mw, phone); err != nil {
						log.Print(err)
					} else if cmd == walk.DlgCmdOK {
						if phone.PhoneEncrypt == "" {
							outTE.SetText("没有输入解密的手机号")
							return
						}
						//手机号码解密
						msg := phoneDecrypt(phone.PhoneEncrypt)
						outTE.SetText(msg)
					}
				},
			},
			PushButton{
				Text:    "文件",
				MinSize: Size{60, 60},
				OnClicked: func() {
					//本机授权
					ip := getLocalIpAddr()
					if ip != "10.1.2.49" {
						walk.MsgBox(mw, "title", "本机没有授权", walk.MsgBoxIconInformation)
						return
					}

					if cmd, err := RunTransferDialog(mw); err != nil {
						walk.MsgBox(mw, "title", err.Error(), walk.MsgBoxIconInformation)
					} else if cmd == walk.DlgCmdOK {
						walk.MsgBox(mw, "title", "Run Transfer Ok.", walk.MsgBoxIconInformation)
					}
				},
			},
			PushButton{
				Text:    "temp文件",
				MinSize: Size{60, 60},
				OnClicked: func() {
					if cmd, err := RunTempFileDialog(mw, tempTextStruct); err != nil {
						walk.MsgBox(mw, "title", err.Error(), walk.MsgBoxIconInformation)
					} else if cmd == walk.DlgCmdOK {
						tempFile, err := sFtpCli.OpenFile("/www/temp.txt", os.O_RDWR|os.O_TRUNC|os.O_CREATE)
						if err != nil {
							walk.MsgBox(mw, "Error", err.Error(), walk.MsgBoxIconError)
						}
						tempFile.Write([]byte(strings.ReplaceAll(tempTextStruct.TempTextString, "\r\n", "\n")))
						tempFile.Close()
					}
				},
			},
			Label{
				Text: "结果:",
				Font: Font{PointSize: 10},
			},
			TextEdit{
				Font:     Font{PointSize: 11},
				VScroll:  true,
				HScroll:  true,
				AssignTo: &outTE,
				ReadOnly: true,
				Text:     fmt.Sprintf("请选择上面需要进行的操作..."),
			},
		},
	}).Create(); err != nil {
		log.Fatal(err)
	}

	// 设置窗体生成在屏幕的正中间
	// 窗体横坐标 = ( 屏幕宽度 - 窗体宽度 ) / 2
	// 窗体纵坐标 = ( 屏幕高度 - 窗体高度 ) / 2
	mw.SetXPixels((int(win.GetSystemMetrics(0))-mw.Width())/2 - 20)
	mw.SetYPixels((int(win.GetSystemMetrics(1)) - mw.Height()) / 2)

	mw.Run()
}

// exeAdress指完整路径
func checkExe2(exeAdress string) {
	cmd := exec.Command("cmd.exe", "/c", "start "+exeAdress)
	err := cmd.Run()
	if err != nil {
		log.Println("启动失败:", err)
	} else {
		log.Println("启动成功!")
	}
}

//获取本机IP地址
func getLocalIpAddr() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

//手机号解密界面
func RunPhoneDecryptDialog(owner walk.Form, phone *Phone) (int, error) {
	var dlg *walk.Dialog
	var db *walk.DataBinder
	var acceptPB, cancelPB *walk.PushButton

	return Dialog{
		AssignTo:      &dlg,
		Title:         "加密手机字符串",
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		DataBinder: DataBinder{
			AssignTo:       &db,
			Name:           "phone",
			DataSource:     phone,
			ErrorPresenter: ToolTipErrorPresenter{},
		},
		MinSize: Size{300, 200},
		Layout:  VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{
						Text: "加密字符串:",
					},
					LineEdit{
						Text: Bind("PhoneEncrypt"),
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &acceptPB,
						Text:     "确定",
						OnClicked: func() {
							if err := db.Submit(); err != nil {
								log.Print(err)
								return
							}
							dlg.Accept()
						},
					},
					PushButton{
						AssignTo:  &cancelPB,
						Text:      "取消",
						OnClicked: func() { dlg.Cancel() },
					},
				},
			},
		},
	}.Run(owner)
}

//手机号码解密
func phoneDecrypt(phoneEncryptString string) string {
	url := "http://cryptserver.com/phone.php?test=0&v=" + phoneEncryptString + "&type=decrypt"
	res, err := http.Get(url)
	if err != nil {
		return "Remote phone encrypt error. err: " + err.Error()
	}
	if res.StatusCode == 200 {
		strByte, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "phone decrypt error. err: " + err.Error()
		} else {
			phoneDecrypt := string(strByte)
			err := clipboard.WriteAll(phoneDecrypt)
			if err != nil {
				return "手机号码: " + phoneDecrypt
			} else {
				return "手机号码: " + phoneDecrypt + "\r\n\r\n已将内容自动赋值到剪切板"
			}
		}
	} else {
		return "Remote phone decrypt url error."
	}
}

//手机号加密界面
func RunPhoneEncryptDialog(owner walk.Form, phone *Phone) (int, error) {
	var dlg *walk.Dialog
	var db *walk.DataBinder
	var acceptPB, cancelPB *walk.PushButton

	return Dialog{
		AssignTo:      &dlg,
		Title:         "手机号码",
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		DataBinder: DataBinder{
			AssignTo:       &db,
			Name:           "phone",
			DataSource:     phone,
			ErrorPresenter: ToolTipErrorPresenter{},
		},
		MinSize: Size{300, 200},
		Layout:  VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{
						Text: "手机号码:",
					},
					LineEdit{
						Text: Bind("PhoneDecrypt"),
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &acceptPB,
						Text:     "确定",
						OnClicked: func() {
							if err := db.Submit(); err != nil {
								log.Print(err)
								return
							}
							dlg.Accept()
						},
					},
					PushButton{
						AssignTo:  &cancelPB,
						Text:      "取消",
						OnClicked: func() { dlg.Cancel() },
					},
				},
			},
		},
	}.Run(owner)
}

//手机号码加密
func phoneEncrypt(phoneDecryptString string) string {
	url := "http://cryptserver.com/phone.php?test=0&v=" + phoneDecryptString + "&type=encrypt"
	res, err := http.Get(url)
	if err != nil {
		return "Remote phone encrypt error."
	}

	if res.StatusCode == 200 {
		strByte, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "phone encrypt error. err: " + err.Error()
		} else {
			phoneEncrypt := string(strByte)
			err := clipboard.WriteAll(phoneEncrypt)
			if err == nil {
				return "加密字符串: " + phoneEncrypt + "\r\n\r\n已将内容自动赋值到剪切板"
			}
		}
	}

	return "Remote phone encrypt url error."
}

//Md5加密界面
func RunMd5EncryptDialog(owner walk.Form, md5str *Md5) (int, error) {
	var dlg *walk.Dialog
	var db *walk.DataBinder
	var acceptPB, cancelPB *walk.PushButton

	return Dialog{
		AssignTo:      &dlg,
		Title:         "MD5加密",
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		DataBinder: DataBinder{
			AssignTo:       &db,
			Name:           "md5str",
			DataSource:     md5str,
			ErrorPresenter: ToolTipErrorPresenter{},
		},
		MinSize: Size{300, 200},
		Layout:  VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					Label{
						Text: "加密字符串:",
					},
					LineEdit{
						Text: Bind("Md5Str"),
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &acceptPB,
						Text:     "确定",
						OnClicked: func() {
							if err := db.Submit(); err != nil {
								log.Print(err)
								return
							}
							dlg.Accept()
						},
					},
					PushButton{
						AssignTo:  &cancelPB,
						Text:      "取消",
						OnClicked: func() { dlg.Cancel() },
					},
				},
			},
		},
	}.Run(owner)
}

//md5加密
func Md5StringEncrypt(md5Str string) string {
	md5EncodeStr := fmt.Sprintf("%x", md5.Sum([]byte(md5Str)))
	err := clipboard.WriteAll(md5EncodeStr)
	if err != nil {
		return "MD5加密字符串: " + md5EncodeStr
	} else {
		return "MD5加密字符串: " + md5EncodeStr + "\r\n\r\n已将内容自动赋值到剪切板"
	}
}

//代码同步界面
func RunCodeRsyncDialog(cli *ssh.Cli, owner walk.Form, codeRsyncItem *CodeRsyncItem) (int, *ssh.Cli, error) {
	var dlg *walk.Dialog
	var db *walk.DataBinder
	var acceptPB, cancelPB *walk.PushButton

	if cli == nil {
		return 0, nil, errors.New("ssh连接失败")
	}

	//第一次初始化，查看目录文件夹
	if len(projectNameArr) == 0 {
		return 0, nil, errors.New("没有发现需要同步的文件夹")
	}

	var radioButtons []RadioButton
	for k, v := range projectNameArr {
		if v == "" {
			continue
		}
		radioButtons = append(radioButtons, RadioButton{
			Text:  v,
			Value: k,
			Font:  Font{PointSize: 10},
		})
		codeRsyncItem.MapItems[k] = v
	}

	num, err := Dialog{
		AssignTo:      &dlg,
		Title:         "选择同步文件夹",
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		DataBinder: DataBinder{
			AssignTo:       &db,
			Name:           "codeRsyncItem",
			DataSource:     codeRsyncItem,
			ErrorPresenter: ToolTipErrorPresenter{},
		},
		MaxSize: Size{360, 640},
		Layout:  VBox{},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 1},
				Children: []Widget{
					RadioButtonGroup{
						DataMember: "CheckItem",
						Buttons:    radioButtons,
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &acceptPB,
						Text:     "确定",
						OnClicked: func() {
							if err := db.Submit(); err != nil {
								log.Print(err)
								return
							}
							dlg.Accept()
						},
					},
					PushButton{
						AssignTo:  &cancelPB,
						Text:      "取消",
						OnClicked: func() { dlg.Cancel() },
					},
				},
			},
		},
	}.Run(owner)

	return num, cli, err
}

//打开传输窗口
func RunTransferDialog(owner walk.Form) (int, error) {
	var dlg *walk.Dialog
	var splitter *walk.Splitter
	var treeView *walk.TreeView
	var logText *walk.TextEdit
	var tableView *walk.TableView
	var acceptPB, cancelPB *walk.PushButton

	treeModel, err := NewDirectoryTreeModel()
	if err != nil {
		log.Fatal(err)
	}
	tableModel := NewFileInfoModel()

	return Dialog{
		AssignTo:      &dlg,
		Title:         "File Transfer",
		Icon:          "favicon.ico",
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		Background: GradientBrush{
			Vertexes: []walk.GradientVertex{
				{X: 0, Y: 0, Color: walk.RGB(255, 255, 127)},
				{X: 1, Y: 0, Color: walk.RGB(127, 191, 255)},
				{X: 0.5, Y: 0.5, Color: walk.RGB(255, 255, 255)},
				{X: 1, Y: 1, Color: walk.RGB(127, 255, 127)},
				{X: 0, Y: 1, Color: walk.RGB(255, 127, 127)},
			},
			Triangles: []walk.GradientTriangle{
				{0, 1, 2},
				{1, 3, 2},
				{3, 4, 2},
				{4, 0, 2},
			},
		},
		MinSize: Size{1230, 680},
		Size:    Size{1230, 680},
		Layout:  VBox{},
		Children: []Widget{
			GroupBox{
				Layout: Grid{Columns: 1},
				Children: []Widget{
					HSplitter{
						Column:   3,
						AssignTo: &splitter,
						Children: []Widget{
							TreeView{
								AssignTo: &treeView,
								Model:    treeModel,
								OnCurrentItemChanged: func() {
									dir := treeView.CurrentItem().(*Directory)
									choosePath := dir.Path()
									logText.AppendText("\r\n" + time.Now().Format("2006-01-02 15:04:05") + " " + choosePath)
									if err := tableModel.SetDirPath(dir.Path()); err != nil {
										walk.MsgBox(
											dlg,
											"Error",
											err.Error(),
											walk.MsgBoxOK|walk.MsgBoxIconError)
									}
								},
							},
							TableView{
								AssignTo:      &tableView,
								StretchFactor: 2,
								Columns: []TableViewColumn{
									TableViewColumn{
										DataMember: "Name",
										Width:      192,
									},
									TableViewColumn{
										DataMember: "Size",
										Format:     "%d",
										Alignment:  AlignFar,
										Width:      64,
									},
									TableViewColumn{
										DataMember: "Modified",
										Format:     "2006-01-02 15:04:05",
										Width:      120,
									},
								},
								Model: tableModel,
							},
							TextEdit{
								Font:     Font{PointSize: 11},
								AssignTo: &logText,
								ReadOnly: true,
								Text:     "Log Record:",
							},
						},
					},
				},
			},
			PushButton{
				Text: "Transfer",
				OnClicked: func() {
					var localDir string

					dir := treeView.CurrentItem().(*Directory)
					localDir = dir.Path()
					if index := tableView.CurrentIndex(); index > -1 {
						name := tableModel.items[index].Name
						localDir = filepath.Join(localDir, name)
					}
					localDir = filepath.ToSlash(localDir)

					localDirArr := strings.Split(localDir, "/")
					localDirArrLen := len(localDirArr)
					name := localDirArr[localDirArrLen-1]
					if name == "" {
						name = localDirArr[localDirArrLen-2]
					}
					remotePath := strings.TrimRight(REMOTE_DIR, "/") + "/" + name

					//上传文件
					//sClient,err := ssh_client.Connect()
					//if err != nil {
					//	walk.MsgBox(dlg,"Error",err.Error(),walk.MsgBoxIconError)
					//}
					msg := ssh_client.Upload(sFtpCli, localDir, remotePath)
					logText.AppendText(msg)
				},
			},
		},
	}.Run(owner)
}

//temp text文件编写界面
func RunTempFileDialog(owner walk.Form, tempTextStruct *TempTextStruct) (int, error) {
	var dlg *walk.Dialog
	var db *walk.DataBinder
	var acceptPB, cancelPB *walk.PushButton
	var tempFileTextEdit *walk.TextEdit

	tempFile, err := sFtpCli.OpenFile("/www/temp.txt", os.O_RDONLY)
	if err != nil {
		walk.MsgBox(dlg, "Error", err.Error(), walk.MsgBoxIconError)
	}
	content, _ := ioutil.ReadAll(tempFile)
	tempFile.Close()

	if err = (Dialog{
		AssignTo:      &dlg,
		Title:         "临时文件存储",
		Icon:          "favicon.ico",
		MinSize:       Size{1200, 800},
		Layout:        VBox{},
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		DataBinder: DataBinder{
			AssignTo:       &db,
			Name:           "tempTextStruct",
			DataSource:     tempTextStruct,
			ErrorPresenter: ToolTipErrorPresenter{},
		},
		Children: []Widget{
			Composite{
				Layout: Grid{Columns: 2},
				Children: []Widget{
					TextEdit{
						Font:     Font{PointSize: 11},
						VScroll:  true,
						HScroll:  true,
						AssignTo: &tempFileTextEdit,
						Text:     Bind("TempTextString"),
					},
				},
			},
			Composite{
				Layout: HBox{},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo: &acceptPB,
						Text:     "保存",
						OnClicked: func() {
							if err := db.Submit(); err != nil {
								log.Print(err)
								return
							}
							dlg.Accept()
						},
					},
					PushButton{
						AssignTo:  &cancelPB,
						Text:      "取消",
						OnClicked: func() { dlg.Cancel() },
					},
				},
			},
		},
	}).Create(owner); err != nil {
		log.Fatal(err)
	}

	tempFileTextEdit.SetText(strings.ReplaceAll(string(content), "\n", "\r\n"))
	tempFileTextEdit.SetFocus()

	return dlg.Run(), err
}
