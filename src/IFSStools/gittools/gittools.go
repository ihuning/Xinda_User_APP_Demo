package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
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

func main() {
	// CloneRepository("https://github.com/iLemonRain/test", "./test", "zc314401480@gmail.com", "zC950303")
	// PushToRepository("./test", "iLemonRain", "zC950303")
	PullFromRepository("./test", "iLemonRain", "zC950303")
}
