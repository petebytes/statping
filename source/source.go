package source

import (
	"errors"
	"fmt"
	"github.com/GeertJohan/go.rice"
	"github.com/hunterlong/statup/utils"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
)

var (
	SqlBox      *rice.Box
	CssBox      *rice.Box
	ScssBox     *rice.Box
	JsBox       *rice.Box
	TmplBox     *rice.Box
	UsingAssets bool
)

func Assets() {
	SqlBox = rice.MustFindBox("sql")
	CssBox = rice.MustFindBox("css")
	ScssBox = rice.MustFindBox("scss")
	JsBox = rice.MustFindBox("js")
	TmplBox = rice.MustFindBox("tmpl")
}

func CompileSASS(folder string) error {
	sassBin := os.Getenv("SASS")
	if sassBin == "" {
		return errors.New("missing the SASS executable environment variable")
	}

	scssFile := fmt.Sprintf("%v/%v", folder, "assets/scss/base.scss")
	baseFile := fmt.Sprintf("%v/%v", folder, "assets/css/base.css")

	utils.Log(1, fmt.Sprintf("Compiling SASS %v into %v", scssFile, baseFile))
	command := fmt.Sprintf("%v %v %v", sassBin, scssFile, baseFile)

	utils.Log(1, fmt.Sprintf("Command: sh -c %v", command))

	testCmd := exec.Command("sh", "-c", command)

	var stdout, stderr []byte
	var errStdout, errStderr error
	stdoutIn, _ := testCmd.StdoutPipe()
	stderrIn, _ := testCmd.StderrPipe()
	testCmd.Start()

	go func() {
		stdout, errStdout = copyAndCapture(os.Stdout, stdoutIn)
	}()

	go func() {
		stderr, errStderr = copyAndCapture(os.Stderr, stderrIn)
	}()

	err := testCmd.Wait()
	if err != nil {
		utils.Log(3, err)
		return err
	}

	if errStdout != nil || errStderr != nil {
		utils.Log(3, fmt.Sprintf("Failed to compile assets with SASS %v", err))
		return errors.New("failed to capture stdout or stderr")
	}

	if err != nil {
		utils.Log(3, fmt.Sprintf("Failed to compile assets with SASS %v", err))
		utils.Log(3, fmt.Sprintf("bash -c %v %v %v", sassBin, scssFile, baseFile))
		return err
	}

	outStr, errStr := string(stdout), string(stderr)
	utils.Log(1, fmt.Sprintf("out: %v | error: %v", outStr, errStr))
	utils.Log(1, "SASS Compiling is complete!")
	return err
}

func HasAssets(folder string) bool {
	if _, err := os.Stat(folder + "/assets"); err == nil {
		utils.Log(1, "Assets folder was found!")
		UsingAssets = true
		return true
	} else {
		assetEnv := os.Getenv("USE_ASSETS")
		if assetEnv == "true" {
			utils.Log(1, "Environment variable USE_ASSETS was found.")
			CreateAllAssets(folder)
			err := CompileSASS(folder)
			if err != nil {
				CopyToPublic(CssBox, folder+"/css", "base.css")
				utils.Log(2, "Default 'base.css' was insert because SASS did not work.")
				return true
			}
			UsingAssets = true
			return true
		}
	}
	return false
}

func SaveAsset(data, folder, file string) {
	utils.Log(1, fmt.Sprintf("Saving %v/%v into assets folder", folder, file))
	err := ioutil.WriteFile(folder+"/assets/"+file, []byte(data), 0644)
	if err != nil {
		utils.Log(3, fmt.Sprintf("Failed to save %v/%v, %v", folder, file, err))
	}
}

func OpenAsset(folder, file string) string {
	dat, err := ioutil.ReadFile(folder + "/assets/" + file)
	if err != nil {
		utils.Log(3, fmt.Sprintf("Failed to open %v, %v", file, err))
		return ""
	}
	return string(dat)
}

func CreateAllAssets(folder string) error {
	utils.Log(1, fmt.Sprintf("Dump Statup assets into %v/assets", folder))
	MakePublicFolder(folder + "/assets")
	MakePublicFolder(folder + "/assets/js")
	MakePublicFolder(folder + "/assets/css")
	MakePublicFolder(folder + "/assets/scss")
	utils.Log(1, "Inserting scss, css, and javascript files into assets folder")
	CopyToPublic(ScssBox, folder+"/assets/scss", "base.scss")
	CopyToPublic(ScssBox, folder+"/assets/scss", "variables.scss")
	CopyToPublic(ScssBox, folder+"/assets/scss", "mobile.scss")
	CopyToPublic(CssBox, folder+"/assets/css", "bootstrap.min.css")
	CopyToPublic(CssBox, folder+"/assets/css", "base.css")
	CopyToPublic(JsBox, folder+"/assets/js", "bootstrap.min.js")
	CopyToPublic(JsBox, folder+"/assets/js", "Chart.bundle.min.js")
	CopyToPublic(JsBox, folder+"/assets/js", "jquery-3.3.1.slim.min.js")
	CopyToPublic(JsBox, folder+"/assets/js", "main.js")
	CopyToPublic(JsBox, folder+"/assets/js", "setup.js")
	CopyToPublic(TmplBox, folder+"/assets", "robots.txt")
	CopyToPublic(TmplBox, folder+"/assets", "statup.png")
	utils.Log(1, "Compiling CSS from SCSS style...")
	err := utils.Log(1, "Statup assets have been inserted")
	return err
}

func DeleteAllAssets(folder string) error {
	err := os.RemoveAll(folder + "/assets")
	if err != nil {
		utils.Log(1, fmt.Sprintf("There was an issue deleting Statup Assets, %v", err))
		return err
	}
	utils.Log(1, "Statup assets have been deleted")
	return err
}

func CopyToPublic(box *rice.Box, folder, file string) {
	assetFolder := fmt.Sprintf("%v/%v", folder, file)
	utils.Log(1, fmt.Sprintf("Copying %v to %v", file, assetFolder))
	base, err := box.String(file)
	if err != nil {
		utils.Log(3, fmt.Sprintf("Failed to copy %v to %v, %v.", file, assetFolder, err))
	}
	err = ioutil.WriteFile(assetFolder, []byte(base), 0644)
	if err != nil {
		utils.Log(3, fmt.Sprintf("Failed to write file %v to %v, %v.", file, assetFolder, err))
	}
}

func MakePublicFolder(folder string) {
	utils.Log(1, fmt.Sprintf("Creating folder '%v'", folder))
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		err = os.MkdirAll(folder, 0755)
		if err != nil {
			utils.Log(3, fmt.Sprintf("Failed to created %v directory, %v", folder, err))
		}
	}
}

func copyAndCapture(w io.Writer, r io.Reader) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1024, 1024)
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			out = append(out, d...)
			_, err := w.Write(d)
			if err != nil {
				return out, err
			}
		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			}
			return out, err
		}
	}
	// never reached
	panic(true)
	return nil, nil
}