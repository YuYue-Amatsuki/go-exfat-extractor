package main

import (
	"fmt"
	"github.com/dsoprea/go-exfat"
	"github.com/dsoprea/go-logging"
	"github.com/jessevdk/go-flags"
	"os"
	"path/filepath"
)

type rootParameters struct {
	FilesystemFilepath string `short:"f" long:"filesystem" description:"Path to the exFAT filesystem image" required:"true"`
	OutputDirectory    string `short:"o" long:"output" description:"Output directory to extract files to" required:"true"`
}

var (
	rootArguments = new(rootParameters)
)

func main() {
	defer func() {
		if state := recover(); state != nil {
			err := log.Wrap(state.(error))
			log.PrintError(err)
			os.Exit(1)
		}
	}()

	p := flags.NewParser(rootArguments, flags.Default)
	_, err := p.Parse()
	if err != nil {
		os.Exit(1)
	}

	// 打开exFAT镜像文件
	f, err := os.Open(rootArguments.FilesystemFilepath)
	log.PanicIf(err)
	defer f.Close()

	// 初始化exFAT解析器
	er := exfat.NewExfatReader(f)
	err = er.Parse()
	log.PanicIf(err)

	// 加载目录树
	tree := exfat.NewTree(er)
	err = tree.Load()
	log.PanicIf(err)

	// 获取所有节点
	files, nodes, err := tree.List()
	log.PanicIf(err)

	// 将相对路径转换为绝对路径
	absoluteOutputDirectory, err := filepath.Abs(rootArguments.OutputDirectory)
	if err != nil {
		fmt.Printf("获取绝对路径时出错: %s\n", err)
		return
	}

	// 遍历所有节点并提取
	for _, currentFilepath := range files {
		node := nodes[currentFilepath]

		fde := node.FileDirectoryEntry()
		isDir := fde.FileAttributes.IsDirectory()

		currentDirPath := ""
		if isDir {
			currentDirPath = currentFilepath
		} else {
			currentDirPath = filepath.Dir(currentFilepath)
		}

		fmt.Printf("Create: %s\n", currentDirPath)
		dstDirPath := filepath.Join(absoluteOutputDirectory, currentDirPath)
		fmt.Printf("Dst: %s\n", dstDirPath)
		err := os.MkdirAll(dstDirPath, 0755)
		if err != nil {
			fmt.Printf("Error creating directory: %s\n", err)
			return
		}

		if !isDir {
			fmt.Printf("Extract: %s\n", currentFilepath)
			dstFilePath := filepath.Join(absoluteOutputDirectory, currentFilepath)
			fmt.Printf("Dst: %s\n", dstFilePath)
			var g *os.File

			g, err = os.Create(dstFilePath)
			log.PanicIf(err)

			defer func() {
				g.Close()
			}()

			sde := node.StreamDirectoryEntry()

			useFat := sde.GeneralSecondaryFlags.NoFatChain() == false

			//clusters, sectors, err := er.WriteFromClusterChain(sde.FirstCluster, sde.ValidDataLength, useFat, g)
			_, _, err = er.WriteFromClusterChain(sde.FirstCluster, sde.ValidDataLength, useFat, g)
			log.PanicIf(err)

			fmt.Printf("(%d) bytes written.\n", sde.ValidDataLength)
			fmt.Printf("\n")
		}

	}
}
