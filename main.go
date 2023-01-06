package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Skeeter struct {
	FondTxt string
	Dir []string
	Files []_file
	Save string
	Scize chan int	`json:"-"`
	Filter []string
	Wa sync.WaitGroup
	StartTime time.Time
}

type _file struct {
	Path string
	Type int
	Val int
	Line int
	OpenTime string
	SousuoTime string
}

var m = map[int]string{
	1 :"相关的目录",
	2 :"相关的文件",
	3 :"过滤列表中搜索关键字在路径中出现",
}


func main(){

	save := flag.String("save","","保存目录")
	text := flag.String("text","","查询的参数")
	filters := flag.String("filter",`!.exe !.rar !.zip !.iso !.vmdk !.vmem !.dll !.vmsn !.pak !.7z`,"过滤不需要的搜索后打开的文件，这意味着还是会搜索但不会打开文件内进一步搜索\n 英文状态下！为不搜索的文件 不加！意味着只包含当前类型。 例子 -filter '!.exe !.zip' ")
	cpunum := flag.Int("cpu",10,"同时启用协程数量")
	dir := flag.String(`dir`,"./",`搜索的目录 默认当前目录内搜索 多目录引号内空格作为分割`+"\n"+`例子 -dir './a ./ ../'`)

	flag.Parse()


	if *text == "" { fmt.Println("请输入正确的命令格式\n-help 获取帮助 不能查询空字符") ; return }
	sk := New(*text,filter(strings.Split(*dir," "))...)
	sk.Save = *save
	sk.Filter = filter(strings.Split(*filters," "))
	sk.Scize = make(chan int,*cpunum)

	fmt.Printf("查询的文本 '%v' 搜索的目录 '%v' 过滤的文件类型：'%v' 协程数量 '%v' 保存目录 '%v'\n",
		sk.FondTxt,
		sk.Dir,
		sk.Filter,
		*cpunum,
		sk.Save,
	)
	sk.StartTime = time.Now()
	sk.foundDir()
	sk.Run()
}
func New(fondTxt string,dirs ...string) *Skeeter {
	var s = &Skeeter{
		FondTxt: fondTxt,
		Dir:   dirs,
		Files: []_file{},
	}
	return s
}

func (sk *Skeeter) Run()  {

	var j int
	for i, file := range sk.Files {
		if file.Type == 1 { continue }
		sk.Wa.Add(1)
		sk.Scize <- i
		num := int(float32(i)/float32(len(sk.Files))*100)
		if j < num {
			j = num
			fmt.Println(j,"%")
		}

		go sk.foundtText(file.Path,sk.FondTxt,i)
	}

	sk.Wa.Wait()

	fmt.Println("100 %")
	fmt.Println("总计耗时：",time.Since(sk.StartTime))

	if sk.Save == "" {fmt.Println("相关路径")}
	for _, file := range sk.Files {
		if file.Type == 0 { continue }
		fmt.Printf(`目录："%v" 第[%v]行 位置："%v" 说明："%v" %v`,file.Path,file.Line,file.Val,m[file.Type],"\n")
	}
	temf := []_file{}
	for _, file := range sk.Files {
		if file.Type > 0 {
			temf = append(temf, file)
		}
	}
	sk.Files = temf
	if sk.Save == "" { sk.Save = "result.json" }
	buf,err := json.MarshalIndent(&sk,"	","")
	if err != nil {
		fmt.Println("保存失败！",err)
		return
	}

	err = os.WriteFile(sk.Save,buf,0666)
	if err != nil {
		fmt.Println("写入结果出现错误：",err)
		return
	}
}

func (sk *Skeeter) foundDir()  {

	for _, s := range sk.Dir {
		err := filepath.Walk(s, func(path string, info fs.FileInfo, err error) error {

			if !info.IsDir() {
				sk.Files = append(sk.Files,_file{Path: path})
			}else {
				if strings.Contains(strings.ToUpper(path),strings.ToUpper(sk.FondTxt)) {
					sk.Files = append(sk.Files,_file{Path: path,Type: 1})
				}
			}
			return nil
		})

		if err != nil { log.Println("get file list err:-" ,s,err) }
	}

}

func (sk *Skeeter) foundtText(path string,args string,i int)  {
	//hello
	defer func() {
		<- sk.Scize
		sk.Wa.Done()
	}()

	for _, s2 := range sk.Filter {
		//fmt.Println(s2[:1])
		if s2[:1] == "!" {
			if strings.Contains(path,s2[1:]) {
				if strings.Contains(path,args) {
					sk.Files[i].Type = 3
					sk.Files[i].Path = path
				}
				return
			}
		} else {
			if !strings.Contains(path,s2) {
				if strings.Contains(path,args) {
					sk.Files[i].Type = 3
					sk.Files[i].Path = path
				}
				return
			}
		}

	}
	st := time.Now()
	//fmt.Println("准备打开文件：",path)
	buf,err := os.ReadFile(path)
	t1 := time.Since(st)

	st1 := time.Now()
	sk.Files[i].OpenTime = t1.String()

	if err != nil { return }

	buf=bytes.ToUpper(buf)

	n:=bytes.Index(buf,bytes.ToUpper([]byte(args)))

	if n != -1 {
		sk.Files[i].Type = 2
		sk.Files[i].Val = n
		for _, b := range buf[:n] {
			if b==10 {
				sk.Files[i].Line ++
			}
		}
		sk.Files[i].Line ++
	}else {
		if strings.Contains(path,args) {
			sk.Files[i].Type = 3
			sk.Files[i].Path = path
		}
	}

	buf = nil
	sk.Files[i].SousuoTime = time.Since(st1).String()
}

func filter(arg []string) (res []string) {

	for _, s := range arg {
		if strings.ReplaceAll(s," ","") == "" { continue }
		res = append(res, s)
	}
	return
}