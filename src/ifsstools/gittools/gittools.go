// git的上传/下载/清空方法.
package gittools

import (
	"fmt"
	"path/filepath"
	"xindauserbackground/src/filetools"
	"xindauserbackground/src/jsontools"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"strings"
)

// git方法的容器
type Git struct {
	UserName string
	Password string
	Url      string
	RepoDir  string
}

// 新建一个git连接
func NewGitClient(url, repoDir, userName, password string) Git {
	g := Git{
		UserName: userName,
		Password: password,
		Url:      url,
		RepoDir:  repoDir,
	}
	return g
}

// 将commit的内容push到在线仓库中
func (g Git) PushToRepository(sendProgressChannel chan []byte) error {
	var err error
	defer func() {
		if err != nil {
			fmt.Println("无法push到仓库", err)
		}
	}()
	// commit过程
	r, err := git.PlainOpen(g.RepoDir)
	if err != nil {
		return err
	}
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	_, fileNameList, err := filetools.GenerateUnhiddenFilePathNameListFromFolder(g.RepoDir)
	if err != nil {
		return err
	}
	for _, fileName := range fileNameList {
		// 将文件存储到暂存区
		_, err = w.Add(fileName)
		if err != nil {
			return err
		} // 填写commit信息并commit
		_, err = w.Commit("", &git.CommitOptions{
			Author: &object.Signature{},
		})
		if err != nil {
			return err
		}
	}
	// 使用默认选项push
	err = r.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: g.UserName,
			Password: g.Password,
		},
	})
	if err != nil {
		// if err.Error() == "command error on refs/heads/master: failed to update ref" {
		// 	for err.Error() == "command error on refs/heads/master: failed to update ref" { // 如果只是在线仓库忙
		// 		err = r.Push(&git.PushOptions{
		// 			Auth: &http.BasicAuth{
		// 				Username: g.UserName,
		// 				Password: g.Password,
		// 			},
		// 		})
		// 		time.Sleep(time.Second)
		// 	}
		// } else {
			return err
		// }
	}
	allFileName := strings.Join(fileNameList, " ")	// 连接所有的fileName,用空格分开
	sendProgressChannelJsonBytes := jsontools.GenerateSendProgressChannelJsonBytes(allFileName, g.Url, g.UserName, len(fileNameList))
	sendProgressChannel <- sendProgressChannelJsonBytes
	return err
}

// 下载仓库中的最新内容,视情况选择clone或者pull
func (g Git) DownloadFromRepository(receiveProgressChannel chan []byte) error {
	var err error
	if !filetools.IsPathExists(filepath.Join(g.RepoDir, ".git")) {
		err = g.CloneRepository()
	} else {
		err = g.PullFromRepository()
	}
	_, fileNameList, err := filetools.GenerateUnhiddenFilePathNameListFromFolder(g.RepoDir)
	if err == nil {
		if len(fileNameList) == 0 {
			fmt.Println("没有在", g.Url, "中检测到需要下载的内容")
			err = fmt.Errorf("没有需要下载的内容")
			return err
		}
		for _, fileName := range fileNameList {
			receiveProgressChannelJsonBytes := jsontools.GenerateReceiveProgressChannelJsonBytes(fileName, g.Url, g.UserName)
			receiveProgressChannel <- receiveProgressChannelJsonBytes
		}
	}
	return err
}

// 将仓库克隆到指定位置
func (g Git) CloneRepository() error {
	var err error
	defer func() {
		if err != nil {
			fmt.Println("无法执行git clone", err)
		}
	}()
	// 如果repoDir已经存在,则需要保护之前的文件,方法为clone空仓库到临时位置,然后移动回来
	isPathExists := filetools.IsPathExists(g.RepoDir)
	if isPathExists {
		tempDir := filepath.Join(g.RepoDir, ".temp")
		_, err = git.PlainClone(tempDir, false, &git.CloneOptions{
			Auth: &http.BasicAuth{
				Username: g.UserName,
				Password: g.Password,
			},
			URL: g.Url,
		})
		err = filetools.RmDir(filepath.Join(g.RepoDir, ".git")) // 如果有这个文件夹要删除
		err = filetools.MoveAllFilesToNewFolder(tempDir, g.RepoDir)
		err = filetools.RmDir(tempDir)
	} else {
		_, err = git.PlainClone(g.RepoDir, false, &git.CloneOptions{
			Auth: &http.BasicAuth{
				Username: g.UserName,
				Password: g.Password,
			},
			URL: g.Url,
		})
		// // 这种方式的目的除了可能是测试clone结果外,还可能是下载数据交换文件的操作,要检查下载了哪些文件.
		// _, fileNameList, err := filetools.GenerateUnhiddenFilePathNameListFromFolder(g.RepoDir)
		// if err == nil {
		// 	for _, fileName := range fileNameList {
		// 		fmt.Println("数据交换文件", fileName, "从", g.Url, "使用git方式成功下载", "使用的账户为", g.UserName)
		// 	}
		// }
	}
	return err
}

// 从在线仓库pull下来
func (g Git) PullFromRepository() error {
	var err error
	defer func() {
		if err != nil {
			if err.Error() == "already up-to-date" {
				return
			}
			fmt.Println("无法从仓库pull", err)
		}
	}()
	// 打开本地仓库
	r, err := git.PlainOpen(g.RepoDir)
	if err != nil {
		return err
	}
	// 获取本地仓库的工作路径
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	// pull操作
	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil {
		return err
	}
	return nil
}

// 清除在线仓库
func (g Git) CleanRepository() error {
	var err error
	defer func() {
		if err != nil && err.Error() != "already up-to-date" {
			fmt.Println("无法clean仓库", err)
		}
	}()
	// tempDir := filepath.Join(g.RepoDir, ".temp")
	// filetools.CopyFolder(filepath.Join(g.RepoDir, ".git"), filepath.Join(g.RepoDir, ".temp", ".git"))
	r, err := git.PlainOpen(g.RepoDir)
	if err != nil {
		return err
	}
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	ref, err := r.Head()
	if err != nil {
		return err
	}
	// 获取所有本地历史上的commits
	cIter, err := r.Log(&git.LogOptions{From: ref.Hash(), All: true})
	if err != nil {
		return err
	}
	var initialHash plumbing.Hash
	err = cIter.ForEach(func(c *object.Commit) error {
		initialHash = c.Hash
		return nil
	})
	if err != nil {
		return err
	}
	err = r.Fetch(&git.FetchOptions{
		RemoteName: "origin",
	})
	if err != nil {
		if err.Error() != "already up-to-date" {
			return err
		} else if err.Error() == "already up-to-date" {
			if err = w.Reset(&git.ResetOptions{
				Mode:   git.HardReset,
				Commit: initialHash,
			}); err != nil {
				return err
			}
			err = r.Push(&git.PushOptions{
				Force: true,
				Auth: &http.BasicAuth{
					Username: g.UserName,
					Password: g.Password,
				}})
			if err == nil || err.Error() == "already up-to-date" {
				if err == nil {
					fmt.Println("git", g.Url, "中的内容已被成功清除", "使用的账户为", g.UserName)
				}
				return nil
			}
			err = filetools.RmDir(g.RepoDir)
		}
	}
	return err
}

// 检查远端是不是有新的commit
func isRepoHasRemoteCommits(r *git.Repository) (bool, error) {
	var err error
	defer func() {
		if err != nil {
			fmt.Println("无法检查远端是不是有新的commit", err)
		}
	}()
	err = r.Fetch(&git.FetchOptions{
		RemoteName: "origin",
	})
	if err != nil {
		return false, err
	}
	remoteLastcommitHash, err := r.ResolveRevision(plumbing.Revision("origin/master"))
	if err != nil {
		return false, err
	}
	localLastcommitHash, err := r.ResolveRevision(plumbing.Revision("master"))
	if err != nil {
		return false, err
	}
	return !(remoteLastcommitHash == localLastcommitHash), err
}
