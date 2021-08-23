package cmd

import (
	"bytes"
	"testing"

	"github.com/fatih/color"
)

func TestManyArgs(t *testing.T) {
	color.NoColor = true
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	ext := Execute(outBuf, errBuf, []string{"00000000000", "000000000000"})
	if ext != Abnormal {
		t.Fatal("failed test1 ManyArgs")
	}
	errMessage := "Error: accepts at most 1 arg(s), received 2\n"
	if errMessage != errBuf.String() {
		t.Fatal("failed test2 ManyArgs")
	}
}

func TestNoArgs(t *testing.T) {
	color.NoColor = true
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	ext := Execute(outBuf, errBuf, []string{})
	if ext != Abnormal {
		t.Fatal("failed test1 NoArgs")
	}
	errMessage := "Error: 伝票番号を入力してください\n"
	if errMessage != errBuf.String() {
		t.Fatal("failed test2 NoArgs")
	}
}

func TestOverSerial(t *testing.T) {
	color.NoColor = true
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	ext := Execute(outBuf, errBuf, []string{"--serial", "11", "00000000000"})
	if ext != Abnormal {
		t.Fatal("failed test1 OvreSerial")
	}
	errMessage := "Error: 連番で取得できるのは 1~10件 までです\n"
	if errMessage != errBuf.String() {
		t.Fatal("failed test2 OverSerial")
	}
}

func TestFewrSerial(t *testing.T) {
	color.NoColor = true
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	ext := Execute(outBuf, errBuf, []string{"--serial", "0", "00000000000"})
	if ext != Abnormal {
		t.Fatal("failed test1 FewSerial")
	}
	errMessage := "Error: 連番で取得できるのは 1~10件 までです\n"
	if errMessage != errBuf.String() {
		t.Fatal("failed test2 FewSerial")
	}
}
