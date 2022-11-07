package build

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// CommandBuild is used to compile proto files
// proto-gen-doc build -o ../doc ../proto
func CommandBuild() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "build",
		Short: "build doc for google protobuf file.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			_, err := ExecuteCommand(args[0], output)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "execute %s args:%v error:%v\n", cmd.Name(), args, err)
				os.Exit(1)
			}

			// _, _ = fmt.Fprintln(os.Stdout, out)
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVarP(&output, "output", "o", output, "output dir")
	return cmd
}

func ExecuteCommand(target, output string) (string, error) {
	var err error

	target, err = filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("failed to abs target path: %w", err)
	}

	output, err = filepath.Abs(output)
	if err != nil {
		return "", fmt.Errorf("failed to abs output path: %w", err)
	}

	var files []string

	// 遍历目录, 合并json文件
	err = filepath.Walk(target, func(path string, info fs.FileInfo, err error) error {
		if strings.HasSuffix(path, ".proto.json") {
			files = append(files, path)
		}

		return err
	})

	var tmpl Template

	err = tmpl.ParseFiles(files...)
	if err != nil {
		return "", fmt.Errorf("parse files failure: %w\n", err)
	}

	renderer := &Renderer{
		tmpl: &tmpl,
	}

	err = renderer.Render(output)
	if err != nil {
		return "", fmt.Errorf("render proto file failure: %w\n", err)
	}

	return "success!", err
}
