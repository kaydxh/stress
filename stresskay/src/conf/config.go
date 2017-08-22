package conf

import (
	"errors"
	//"fmt"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
)

type Conf struct {
	curPtr  *map[string]string
	basic   map[string]string
	section map[string]map[string]string
}

func NewConf() *Conf {
	return &Conf{
		basic:   make(map[string]string),
		section: make(map[string]map[string]string),
	}
}

type sectionConf struct {
	closed bool
	name   string
	data   map[string]string
}

func initSectionConf() *sectionConf {
	return &sectionConf{
		closed: true,
		name:   "",
		data:   make(map[string]string),
	}
}

func (self *Conf) GetSectionNum() int {
	return len(self.section)
}

func (self *Conf) LoadFile(path string) (err error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	sectionConf := initSectionConf()
	dataSlice := strings.Split(string(data), "\n")
	for ln, line := range dataSlice {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' || len(line) <= 3 {
			continue
		}

		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return errors.New("line " + strconv.Itoa(ln) + ": invalid config syntax")

			}

			if !sectionConf.closed {
				self.section[sectionConf.name] = sectionConf.data
			}

			sectionConf = initSectionConf()
			sectionConf.closed = false
			sectionConf.name = strings.Trim(line, "[]")
			continue
		}

		lineSlice := strings.SplitN(line, "=", 2)
		if len(lineSlice) != 2 {
			return errors.New("line " + strconv.Itoa(ln) + ": invalid config syntax")
		}

		k := strings.TrimSpace(lineSlice[0])
		v := strings.TrimSpace(lineSlice[1])
		if !sectionConf.closed {
			sectionConf.data[k] = v
		} else {
			self.basic[k] = v
		}
	}

	if !sectionConf.closed {
		self.section[sectionConf.name] = sectionConf.data
	}

	//fmt.Println("sectionConf ALL: ", self)

	return nil
}

func (self *Conf) Parse(obj interface{}) error {
	objT := reflect.TypeOf(obj)
	eT := objT.Elem()
	//fmt.Println("objT: ", objT, ", eT: ", eT)
	if objT.Kind() != reflect.Ptr || eT.Kind() != reflect.Struct {
		return errors.New("obj must be pointer to struct")
	}
	objV := reflect.ValueOf(obj)
	eV := objV.Elem()

	self.curPtr = &self.basic

	self.parseField(eT, eV)

	return nil
}

func (self *Conf) parseField(eT reflect.Type, eV reflect.Value) {
	for i := 0; i < eT.NumField(); i++ {
		f := eT.Field(i)
		t := string(f.Tag)
		if t == "" {
			t = f.Name
		}

		fV := eV.Field(i)
		if !fV.CanSet() {
			continue
		}

		switch f.Type.Kind() {
		case reflect.Bool:
			if v, b := self.getItemBool(t); b {
				fV.SetBool(v)
			}

		case reflect.Int:
			if v, b := self.getItemInt(t); b {
				fV.SetInt(int64(v))
			}

		case reflect.String:
			if v, b := self.getItemString(t); b {
				fV.SetString(v)
			}

		case reflect.Slice:
			eT2 := f.Type.Elem()
			if eT2.Kind() != reflect.Struct {
				continue
			}

			fallthrough

		case reflect.Array:
			eT2 := eT
			eV2 := eV
			idx := 0

			for k := range self.section {
				sec, _ := self.section[k]
				self.curPtr = &sec
				t = k

				eT = f.Type.Elem()
				eV = fV.Index(idx)
				self.parseField(eT, eV)
				idx++

			}

			eT = eT2
			eV = eV2

		default:
		}
	}
}

func (self *Conf) getItemBool(name string) (bool, bool) {
	val := (*self.curPtr)[name]
	if val == "" {
		return false, false
	}

	v, _ := strconv.ParseBool(strings.ToLower(val))
	return v, true
}

func (self *Conf) getItemInt(name string) (int, bool) {
	val, exist := (*self.curPtr)[name]
	if !exist {
		return 0, false
	}

	v, err := strconv.Atoi(val)
	if err != nil {
		return 0, false
	}

	return v, true
}

func (self *Conf) getItemInt64(name string) (int64, bool) {
	val, exist := (*self.curPtr)[name]
	if !exist {
		return 0, false
	}

	v, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, false
	}

	return v, true
}

func (self *Conf) getItemString(name string) (string, bool) {
	val, exist := (*self.curPtr)[name]
	return val, exist
}
