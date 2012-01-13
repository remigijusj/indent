package main

import (
  "bytes"
  "fmt"
  "io/ioutil"
  "testing"
)

var testFiles = []string{"test/t1", "test/t2"}

func TestMissingFile(t *testing.T) {
  arg := "missing"
  err := handleFile(arg)
  fmt.Println(arg, err)
  if err == nil { t.Fail() }
}

func TestTestFiles(t *testing.T) {
  *debug = true
  
  for _,arg := range testFiles {
    err := handleFile(arg)
    if err != nil { t.Fail() }
    
    eql, err := filesEqual(arg+".indent", arg+".result")
    if err != nil { t.Fail() }
    if !eql { t.Fail() }
  }
}

func filesEqual(f1, f2 string) (eql bool, err error) {
  data1, err := ioutil.ReadFile(f1)
  if err != nil { return }
  
  data2, err := ioutil.ReadFile(f2)
  if err != nil { return }
  
  eql = bytes.Compare(data1, data2) == 0
  return
}
