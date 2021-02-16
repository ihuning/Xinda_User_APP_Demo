package gittools

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// 将仓库克隆到指定位置
func CloneRepository(url, directory, username, password string) error {
	r, err := git.PlainClone(directory, false, &git.CloneOptions{
		Auth: &http.BasicAuth{
			Username: username,
			Password: password,
		},
		URL: url,
	})
	if err != nil {
		fmt.Println(err)
		return err
	}
	// 检索HEAD指向的分支
	ref, err := r.Head()
	if err != nil {
		fmt.Println(err)
		return err
	}
	// 计算Hash
	_, err = r.CommitObject(ref.Hash())
	if err != nil {
		fmt.Println(err)
		return err
	}
	fmt.Printf("将仓库 %s 克隆到 %s 成功\n", url, directory)
	return nil
}

// 将commit的内容push到在线仓库中
func PushToRepository(directory, username, password string) error {
	// commit过程
	r, err := git.PlainOpen(directory)
	if err != nil {
		return err
	}
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	// 生成一个新文件
	filename := time.Now().Format("2006-01-02 15:04:05")
	err = ioutil.WriteFile(filepath.Join(directory, filename), []byte("hello world"), 0644)
	if err != nil {
		return err
	}
	// 将文件存储到暂存区
	_, err = w.Add(filename)
	if err != nil {
		return err
	}
	// 验证worktree当前状态
	_, err = w.Status()
	if err != nil {
		return err
	}
	// 填写commit信息并commit
	_, err = w.Commit("commit信息", &git.CommitOptions{
		Author: &object.Signature{},
	})
	if err != nil {
		return err
	}
	// push过程
	d, err := git.PlainOpen(directory)
	if err != nil {
		return err
	}
	// 使用默认选项push
	err = d.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: username,
			Password: password,
		},
	})
	if err != nil {
		return err
	}
	fmt.Printf("将位置 %s push成功\n", directory)
	return nil
}

// 将commit的内容push到在线仓库中
func PullFromRepository(directory, username, password string) error {
	var err error
	defer func() {
		if err != nil {
			fmt.Println("无法push到仓库", err)
		}
	}()
	// 打开本地仓库
	r, err := git.PlainOpen(directory)
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
	fmt.Printf("从位置 %s pull成功\n", directory)
	return nil
}

func ResetLastCommit(directory, username, password string) error {
	var err error
	r, err := git.PlainOpen(directory)
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
	cIter, err := r.Log(&git.LogOptions{From: ref.Hash(), All:true})
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
