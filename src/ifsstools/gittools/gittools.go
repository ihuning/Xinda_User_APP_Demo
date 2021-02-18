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

// 将仓库克隆到指定位置
func CloneRepository(url, repoDir, username, password string) error {
	_, err := git.PlainClone(repoDir, false, &git.CloneOptions{
		Auth: &http.BasicAuth{
			Username: username,
			Password: password,
		},
		URL: url,
	})
	if err != nil {
		// 如果远程仓库未初始化,应该创立之初就建立一个包含.gitignore的空仓库
		fmt.Println("无法克隆仓库")
		return err
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
	filePathList, err := filetools.GenerateFilePathListFromFolder(repoDir)
	if err != nil {
		return err
	}
	for _, filePath := range filePathList {
		// 将文件存储到暂存区
		_, fileName := filepath.Split(filePath)
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
