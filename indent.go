package main

import (
  "bufio"
  "bytes"
  "errors"
  "flag"
  "fmt"
  "os"
  "io/ioutil"
  "strconv"
)

const cap_line = 50            // initial capacity of lines slice
const max_size = 1 << (2 * 10) // 1 Mb
const crlf = "\r\n"            // Windows

var debug     = flag.Bool("d", false, "debug info, don't overwrite files")
var tab_width = flag.Int("t", 2, "tab 'width', for conversion to spaces")

type lineInfo struct {
  indent int
  body   []byte
}

func main() {
  flag.Parse()
  
  for _,arg := range flag.Args() {
    if *debug {
      fmt.Printf("[%s]\n", arg)
    }
    err := handleFile(arg)
    if err != nil {
      fmt.Println(arg, err)
    }
  }
}

func handleFile(filename string) error {
  // prevent big files
  fi, err := os.Stat(filename)
  if err != nil { return err }
  if fi.Size() > max_size { return errors.New("file too big") }
  
  // slurp source file
  data, err := ioutil.ReadFile(filename)
  if err != nil { return err }
  
  tempfile := filename+".indent"
  
  // prepare buffer-write to temp file
  f, err := os.OpenFile(tempfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0)
  if err != nil { return err }
  buf := bufio.NewWriter(f) // <<< ..Size?
  
  // main processing and flush
  lines := readLines(data)
  reindent(lines)
  err = writeLines(lines, buf)
  
  // finalize writing
  f.Close()
  if err != nil {
    os.Remove(tempfile)
    return err
  }
  
  // when debugging, don't overwrite
  if *debug { return nil }
  
  // replace original file?
  err = os.Remove(filename)
  if err != nil { return err } // leaving tempfile!
    
  err = os.Rename(tempfile, filename)
  if err != nil { return err } // leaving tempfile!
  
  return nil
}

func readLines(in []byte) []*lineInfo {
  var cnt, lend, size, offset, offset2 int
  var line_start, line_stop, line_tabs, line_indent int
  var line []byte
  
  lines := make([]*lineInfo, 0, cap_line)
  
  // read lines, tabs -> spaces
  for {
    cnt++
    
    lend = bytes.IndexByte(in[offset:], '\n') // end marker
    size = lend + 1
    if lend < 0 { // EOF without CRLF
      size = len(in) - offset
    }
    if size == 0 { break } // EOF just after CRLF
    
    offset2 = offset + size
    line = in[offset:offset2]
    
    line_start, line_stop, line_tabs = lineMarkers(line)
    
    line_indent = line_start + (*tab_width - 1) * line_tabs // tab to spaces
    
    lines = append(lines, &lineInfo{indent: line_indent, body: line[line_start:line_stop]})
    
    if *debug {
      fmt.Printf("%3d: size=%-2d %2d:%-2d ind=%-2d %s\n", 
                  cnt, size, line_start, line_stop, line_indent, strconv.Quote(string(line)))
    }
    
    if lend < 0 { break }
    offset = offset2
  }
  
  return lines
}

func reindent(lines []*lineInfo) {
  var prev, next *lineInfo
  
  fake := lineInfo{indent: 0, body: make([]byte, 1)} // not empty
  
  for idx, item := range lines {
    // adjust indent of empty line
    if len(item.body) == 0 {
      if idx > 0 {
        prev = lines[idx-1]
      }
      
      // find next non-empty
      next = &fake
      for jdx := idx+1; jdx < len(lines); jdx++ {
        if len(lines[jdx].body) > 0 {
          next = lines[jdx]
          break
        }
      }
      
      // snap it to prev or next <<< simplify?
      switch {
      case len(prev.body) == 0: // align to prev empty
        item.indent = prev.indent
        
      case item.indent < prev.indent && item.indent < next.indent: // pull up
        if prev.indent < next.indent {
          item.indent = prev.indent
        } else {
          item.indent = next.indent
        }
        
      case item.indent > next.indent && item.indent > next.indent: // pull down
        if prev.indent < next.indent {
          item.indent = next.indent
        } else {
          item.indent = prev.indent
        }
        
      case prev.indent < next.indent:
        if float32(item.indent - prev.indent)/float32(next.indent - prev.indent) <= 0.5 {
          item.indent = prev.indent
        } else {
          item.indent = next.indent
        }
        
      case prev.indent > next.indent:
        if float32(item.indent - next.indent)/float32(prev.indent - next.indent) <= 0.5 {
          item.indent = next.indent
        } else {
          item.indent = prev.indent
        }
      }
    }
  }
}

func writeLines(lines []*lineInfo, out *bufio.Writer) (err error) {
  var size int
  var line []byte
  
  for _, item := range lines {
    // prepare the line
    size = item.indent + len(item.body)
    line = make([]byte, size+len(crlf))
    
    for i := 0; i < item.indent; i++ {
      line[i] = ' ' // proper indent
    }
    copy(line[item.indent:], item.body)
    copy(line[size:],        crlf)
    
    // write to buffer
    _, err = out.Write(line)
    if err != nil { return err }
  }
  
  err = out.Flush()
  return
}

func lineMarkers(line []byte) (start int, stop int, tabs int) {
  // left scan spaces
  start = 0
  S: for start < len(line) {
    switch line[start] {
    case '\t':
      tabs++
    case ' ':
    default:
      break S
    }
    start++
  }
  
  // right scan spaces
  stop = len(line)
  E: for stop > start {
    switch line[stop-1] {
    case '\n', '\r', ' ', '\t', '\f':
    default:
      break E
    }
    stop--
  }
  
  return
}
