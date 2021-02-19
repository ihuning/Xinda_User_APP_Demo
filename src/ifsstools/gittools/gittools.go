package gittools

import (
	"fmt"
	"path/filepath"
	"xindauserbackground/src/filetools"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

func TestGitConnection(url, username, password string) error {
	var err error
	defer func() {
		if err != nil {
			fmt.Println("git远程仓库连接测试失败", err)
		}
	}()
	tempDir := "./.testGitConnection"
	err = CloneRepository(url, tempDir, username, password)
	err = filetools.Rmdir(tempDir)
	return err
}

// 将仓库克隆到指定位置
func CloneRepository(url, repoDir, username, password string) error {
	var err error
	defer func() {
		if err != nil {
			fmt.Println("git clone失败", err)
		}
	}()
	// 如果repoDir已经存在,则需要保护之前的文件,方法为clone空仓库到临时位置,然后移动回来
	isPathExists := filetools.IsPathExists(repoDir)
	if isPathExists {
		tempDir := filepath.Join(repoDir, ".temp")
		_, err = git.PlainClone(tempDir, false, &git.CloneOptions{
			Auth: &http.BasicAuth{
				Username: username,
				Password: password,
			},
			URL: url,
		})
		filePathList, fileNameList, _ := filetools.GenerateAllFilePathNameListFromFolder(tempDir)
		for i := 0; i < len(filePathList); i++ {
			err = filetools.Rename(filePathList[i], filepath.Join(repoDir, fileNameList[i]))
		}
		err = filetools.Rmdir(tempDir)
	} else {
		_, err = git.PlainClone(repoDir, false, &git.CloneOptions{
			Auth: &http.BasicAuth{
				Username: username,
				Password: password,
			},
			URL: url,
		})
	}
	return err
}

// 将commit的内容push到在线仓库中
func PushToRepository(repoDir, username, password string) error {
	var err error
	defer func() {
		if err != nil {
			fmt.Println("无法push到仓库", err)
		}
	}()
	// commit过程
	r, err := git.PlainOpen(repoDir)
	if err != nil {
		return err
	}
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	_, fileNameList, err := filetools.GenerateSpecFilePathNameListFromFolder(repoDir)
	if err != nil {
		return err
	}
	for _, fileName := range fileNameList {
		// 将文件存储到暂存区
		_, err = w.Add(fileName)
		if err != nil {
			return err
		}
	}
	// 填写commit信息并commit
	_, err = w.Commit("", &git.CommitOptions{
		Author: &object.Signature{},
	})
	if err != nil {
		return err
	}
	// 使用默认选项push
	err = r.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: username,
			Password: password,
		},
	})
	return err
}

// 从在线仓库pull下来
func PullFromRepository(repoDir, username, password string) error {
	var err error
	defer func() {
		if err != nil {
			fmt.Println("无法从仓库pull", err)
		}
	}()
	// 打开本地仓库
	r, err := git.PlainOpen(repoDir)
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

func CleanRepository(repoDir, username, password string) error {
	var err error
	defer func() {
		if err != nil {
			fmt.Println("无法clean仓库", err)
		}
	}()
	r, err := git.PlainOpen(repoDir)
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
	// 获取所有历史上的commits
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
	if err = w.Reset(&git.ResetOptions{
		Mode:   git.HardReset,
		Commit: initialHash,
	}); err != nil {
		return err
	}
	err = r.Push(&git.PushOptions{
		Force: true,
		Auth: &http.BasicAuth{
			Username: username,
			Password: password,
		}})
	return err
}
