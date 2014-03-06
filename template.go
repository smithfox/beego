package beego

//@todo add template funcs

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	beegoTplFuncMap template.FuncMap
	BeeTemplates    map[string]*template.Template
	BeeTemplateExt  []string
)

func init() {
	BeeTemplates = make(map[string]*template.Template)
	beegoTplFuncMap = make(template.FuncMap)
	BeeTemplateExt = make([]string, 0)
	BeeTemplateExt = append(BeeTemplateExt, "tpl", "html")
	beegoTplFuncMap["dateformat"] = DateFormat
	beegoTplFuncMap["date"] = Date
	beegoTplFuncMap["compare"] = Compare
	beegoTplFuncMap["substr"] = Substr
	beegoTplFuncMap["html2str"] = Html2str
	beegoTplFuncMap["str2html"] = Str2html
	beegoTplFuncMap["htmlquote"] = Htmlquote
	beegoTplFuncMap["htmlunquote"] = Htmlunquote
	beegoTplFuncMap["renderform"] = RenderForm
	/*
		// go1.2 added template funcs
		// Comparisons
		beegoTplFuncMap["eq"] = eq // ==
		beegoTplFuncMap["ge"] = ge // >=
		beegoTplFuncMap["gt"] = gt // >
		beegoTplFuncMap["le"] = le // <=
		beegoTplFuncMap["lt"] = lt // <
		beegoTplFuncMap["ne"] = ne // !=

		beegoTplFuncMap["urlfor"] = UrlFor // !=
	*/
}

// AddFuncMap let user to register a func in the template
func AddFuncMap(key string, funname interface{}) error {
	if _, ok := beegoTplFuncMap[key]; ok {
		return errors.New("funcmap already has the key")
	}
	beegoTplFuncMap[key] = funname
	return nil
}

type templatefile struct {
	root  string
	files []string
}

func (self *templatefile) visit(paths string, f os.FileInfo, err error) error {
	if f == nil {
		return err
	}
	if f.IsDir() || (f.Mode()&os.ModeSymlink) > 0 {
		return nil
	}

	if !HasTemplateExt(paths) {
		return nil
	}

	replace := strings.NewReplacer("\\", "/")
	a := []byte(paths)
	a = a[len([]byte(self.root)):]
	file := strings.TrimLeft(replace.Replace(string(a)), "/")

	self.files = append(self.files, file)

	return nil
}

func HasTemplateExt(paths string) bool {
	for _, v := range BeeTemplateExt {
		if strings.HasSuffix(paths, "."+v) {
			return true
		}
	}
	return false
}

func AddTemplateExt(ext string) {
	for _, v := range BeeTemplateExt {
		if v == ext {
			return
		}
	}
	BeeTemplateExt = append(BeeTemplateExt, ext)
}

func BuildTemplate(dir string) error {
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return errors.New("dir open err")
		}
	}
	self := &templatefile{
		root:  dir,
		files: make([]string, 0),
	}
	err := filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		return self.visit(path, f, err)
	})
	if err != nil {
		fmt.Printf("filepath.Walk() returned %v\n", err)
		return err
	}
	for _, file := range self.files {

		t, err := getTemplate(self.root, file, self.files...)
		if err != nil {
			fmt.Printf("Parse Template %s err=%v\n", file, err)
		} else {
			BeeTemplates[file] = t
		}

	}
	return nil
}

func getTplDeep(root, file, parent string, t *template.Template) (*template.Template, [][]string, error) {
	var fileabspath string
	if filepath.HasPrefix(file, "../") {
		fileabspath = filepath.Join(root, filepath.Dir(parent), file)
	} else {
		fileabspath = filepath.Join(root, file)
	}
	if bb, _ := FileExists(fileabspath); !bb {
		panic("can't find template file" + file)
	}
	data, err := ioutil.ReadFile(fileabspath)
	if err != nil {
		return nil, [][]string{}, err
	}
	t, err = t.New(file).Parse(string(data))
	if err != nil {
		return nil, [][]string{}, err
	}
	reg := regexp.MustCompile(TemplateLeft + "[ ]*template[ ]+\"([^\"]+)\"")
	allsub := reg.FindAllStringSubmatch(string(data), -1)
	for _, m := range allsub {
		if len(m) == 2 {
			tlook := t.Lookup(m[1])
			if tlook != nil {
				continue
			}
			if !HasTemplateExt(m[1]) {
				continue
			}
			t, _, err = getTplDeep(root, m[1], file, t)
			if err != nil {
				return nil, [][]string{}, err
			}
		}
	}
	return t, allsub, nil
}

func getTemplate(root, file string, others ...string) (t *template.Template, err error) {
	t = template.New(file).Delims(TemplateLeft, TemplateRight).Funcs(beegoTplFuncMap)
	var submods [][]string
	t, submods, err = getTplDeep(root, file, "", t)
	if err != nil {
		return nil, err
	}
	t, err = _getTemplate(t, root, submods, others...)

	if err != nil {
		return nil, err
	}
	return
}

func _getTemplate(t0 *template.Template, root string, submods [][]string, others ...string) (t *template.Template, err error) {
	t = t0
	for _, m := range submods {
		if len(m) == 2 {
			templ := t.Lookup(m[1])
			if templ != nil {
				continue
			}
			//first check filename
			for _, otherfile := range others {
				if otherfile == m[1] {
					var submods1 [][]string
					t, submods1, err = getTplDeep(root, otherfile, "", t)
					if err != nil {
						fmt.Printf("Beego Template getTplDeep %s err=%v\n", otherfile, err)
					} else if submods1 != nil && len(submods1) > 0 {
						t, err = _getTemplate(t, root, submods1, others...)
					}
					break
				}
			}
			//second check define
			for _, otherfile := range others {
				fileabspath := filepath.Join(root, otherfile)
				data, err := ioutil.ReadFile(fileabspath)
				if err != nil {
					continue
				}
				reg := regexp.MustCompile(TemplateLeft + "[ ]*define[ ]+\"([^\"]+)\"")
				allsub := reg.FindAllStringSubmatch(string(data), -1)
				for _, sub := range allsub {
					if len(sub) == 2 && sub[1] == m[1] {
						var submods1 [][]string
						t, submods1, err = getTplDeep(root, otherfile, "", t)
						if err != nil {
							fmt.Printf("Beego Template getTplDeep %s err=%v\n", otherfile, err)
						} else if submods1 != nil && len(submods1) > 0 {
							t, err = _getTemplate(t, root, submods1, others...)
						}
						break
					}
				}
			}
		}

	}
	return
}
