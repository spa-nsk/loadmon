package csvtools

import (
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	formatTime        = "15:04:05"
	formatDate        = "2006-01-02"
	formatDateRus     = "02.01.2006"
	formatDateTime    = "2006-01-02 15:04:05"
	formatDateTimeT   = "2006-01-02T15:04:05"
	formatDateTimeRus = "02.01.2006 15:04:05"
)

// Substr returns the substr from start to length.
func Substr(s string, start, length int) string {
	bt := []rune(s)
	if start < 0 {
		start = 0
	}
	if start > len(bt) {
		start = start % len(bt)
	}
	var end int
	if (start + length) > (len(bt) - 1) {
		end = len(bt)
	} else {
		end = start + length
	}
	return string(bt[start:end])
}

// DateFormat takes a time and a layout string and returns a string with the formatted date. Used by the template parser as "dateformat"
func DateFormat(t time.Time, layout string) (datestring string) {
	datestring = t.Format(layout)
	return
}

// DateFormat pattern rules.
var datePatterns = []string{
	// year
	"Y", "2006", // A full numeric representation of a year, 4 digits  Examples: 1999 or 2003
	"y", "06", // A two digit representation of a year  Examples: 99 or 03

	// month
	"m", "01", // Numeric representation of a month, with leading zeros 01 through 12
	"n", "1", // Numeric representation of a month, without leading zeros  1 through 12
	"M", "Jan", // A short textual representation of a month, three letters Jan through Dec
	"F", "January", // A full textual representation of a month, such as January or March  January through December

	// day
	"d", "02", // Day of the month, 2 digits with leading zeros 01 to 31
	"j", "2", // Day of the month without leading zeros 1 to 31

	// week
	"D", "Mon", // A textual representation of a day, three letters Mon through Sun
	"l", "Monday", // A full textual representation of the day of the week  Sunday through Saturday

	// time
	"g", "3", // 12-hour format of an hour without leading zeros  1 through 12
	"G", "15", // 24-hour format of an hour without leading zeros  0 through 23
	"h", "03", // 12-hour format of an hour with leading zeros  01 through 12
	"H", "15", // 24-hour format of an hour with leading zeros  00 through 23

	"a", "pm", // Lowercase Ante meridiem and Post meridiem am or pm
	"A", "PM", // Uppercase Ante meridiem and Post meridiem AM or PM

	"i", "04", // Minutes with leading zeros  00 to 59
	"s", "05", // Seconds, with leading zeros  00 through 59

	// time zone
	"T", "MST",
	"P", "-07:00",
	"O", "-0700",

	// RFC 2822
	"r", time.RFC1123Z,
}

// DateParse Parse Date use PHP time format.
func DateParse(dateString, format string) (time.Time, error) {
	replacer := strings.NewReplacer(datePatterns...)
	format = replacer.Replace(format)
	return time.ParseInLocation(format, dateString, time.Local)
}

// Date takes a PHP like date func to Go's time format.
func Date(t time.Time, format string) string {
	replacer := strings.NewReplacer(datePatterns...)
	format = replacer.Replace(format)
	return t.Format(format)
}

// Compare is a quick and dirty comparison function. It will convert whatever you give it to strings and see if the two values are equal.
// Whitespace is trimmed. Used by the template parser as "eq".
func Compare(a, b interface{}) (equal bool) {
	equal = false
	if strings.TrimSpace(fmt.Sprintf("%v", a)) == strings.TrimSpace(fmt.Sprintf("%v", b)) {
		equal = true
	}
	return
}

// CompareNot !Compare
func CompareNot(a, b interface{}) (equal bool) {
	return !Compare(a, b)
}

// NotNil the same as CompareNot
func NotNil(a interface{}) (isNil bool) {
	return CompareNot(a, nil)
}

var sliceOfInts = reflect.TypeOf([]int(nil))
var sliceOfStrings = reflect.TypeOf([]string(nil))

var unKind = map[reflect.Kind]bool{
	reflect.Uintptr:       true,
	reflect.Complex64:     true,
	reflect.Complex128:    true,
	reflect.Array:         true,
	reflect.Chan:          true,
	reflect.Func:          true,
	reflect.Map:           true,
	reflect.Ptr:           true,
	reflect.Slice:         true,
	reflect.Struct:        true,
	reflect.UnsafePointer: true,
}

func isStructPtr(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct
}

// parseFormTag takes the stuct-tag of a StructField and parses the 'csv' value.
// returned are the form label, name-property, type and wether the field should be ignored.
func parseFormTag(fieldT reflect.StructField) (label, name, fType string, id string, class string, ignored bool, required bool) {
	tags := strings.Split(fieldT.Tag.Get("csv"), ",")
	label = fieldT.Name + ":"
	name = fieldT.Name
	fType = "text"
	ignored = false
	id = fieldT.Tag.Get("id")
	class = fieldT.Tag.Get("class")

	required = false
	requiredField := fieldT.Tag.Get("required")
	if requiredField != "-" && requiredField != "" {
		required, _ = strconv.ParseBool(requiredField)
	}

	switch len(tags) {
	case 1:
		if tags[0] == "-" {
			ignored = true
		}
		if len(tags[0]) > 0 {
			name = tags[0]
		}
	case 2:
		if len(tags[0]) > 0 {
			name = tags[0]
		}
		if len(tags[1]) > 0 {
			fType = tags[1]
		}
	case 3:
		if len(tags[0]) > 0 {
			name = tags[0]
		}
		if len(tags[1]) > 0 {
			fType = tags[1]
		}
		if len(tags[2]) > 0 {
			label = tags[2]
		}
	}

	return
}

// ParseForm will parse form values to struct via tag.
// Support for anonymous struct.
func parseFormToStruct(form url.Values, objT reflect.Type, objV reflect.Value) error {
	for i := 0; i < objT.NumField(); i++ {
		fieldV := objV.Field(i)
		if !fieldV.CanSet() {
			continue
		}

		fieldT := objT.Field(i)
		if fieldT.Anonymous && fieldT.Type.Kind() == reflect.Struct {
			err := parseFormToStruct(form, fieldT.Type, fieldV)
			if err != nil {
				return err
			}
			continue
		}

		tags := strings.Split(fieldT.Tag.Get("csv"), ",")
		var tag string
		if len(tags) == 0 || len(tags[0]) == 0 {
			tag = fieldT.Name
		} else if tags[0] == "-" {
			continue
		} else {
			tag = tags[0]
		}

		formValues := form[tag]
		var value string
		if len(formValues) == 0 {
			defaultValue := fieldT.Tag.Get("default")
			if defaultValue != "" {
				value = defaultValue
			} else {
				continue
			}
		}
		if len(formValues) == 1 {
			value = formValues[0]
			if value == "" {
				// _ = value
				continue
			}
		}

		switch fieldT.Type.Kind() {
		case reflect.Bool:
			if strings.ToLower(value) == "on" || strings.ToLower(value) == "1" || strings.ToLower(value) == "yes" {
				fieldV.SetBool(true)
				continue
			}
			if strings.ToLower(value) == "off" || strings.ToLower(value) == "0" || strings.ToLower(value) == "no" {
				fieldV.SetBool(false)
				continue
			}
			b, err := strconv.ParseBool(value)
			if err != nil {
				return err
			}
			fieldV.SetBool(b)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			x, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			fieldV.SetInt(x)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			x, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return err
			}
			fieldV.SetUint(x)
		case reflect.Float32, reflect.Float64:
			x, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return err
			}
			fieldV.SetFloat(x)
		case reflect.Interface:
			fieldV.Set(reflect.ValueOf(value))
		case reflect.String:
			fieldV.SetString(value)
		case reflect.Struct:
			switch fieldT.Type.String() {
			case "time.Time":
				var (
					t   time.Time
					err error
				)
				if len(value) >= 25 {
					value = value[:25]
					t, err = time.ParseInLocation(time.RFC3339, value, time.Local)
				} else if strings.HasSuffix(strings.ToUpper(value), "Z") {
					t, err = time.ParseInLocation(time.RFC3339, value, time.Local)
				} else if len(value) >= 19 {
					if strings.Contains(value, "T") {
						value = value[:19]
						t, err = time.ParseInLocation(formatDateTimeT, value, time.Local)
					} else {
						value = value[:19]
						t, err = time.ParseInLocation(formatDateTime, value, time.Local)
					}
				} else if len(value) >= 10 {
					if len(value) > 10 {
						value = value[:10]
					}
					t, err = time.ParseInLocation(formatDate, value, time.Local)
				} else if len(value) >= 8 {
					if len(value) > 8 {
						value = value[:8]
					}
					t, err = time.ParseInLocation(formatTime, value, time.Local)
				}
				if err != nil {
					return err
				}
				fieldV.Set(reflect.ValueOf(t))
			}
		case reflect.Slice:
			if fieldT.Type == sliceOfInts {
				formVals := form[tag]
				fieldV.Set(reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(int(1))), len(formVals), len(formVals)))
				for i := 0; i < len(formVals); i++ {
					val, err := strconv.Atoi(formVals[i])
					if err != nil {
						return err
					}
					fieldV.Index(i).SetInt(int64(val))
				}
			} else if fieldT.Type == sliceOfStrings {
				formVals := form[tag]
				fieldV.Set(reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf("")), len(formVals), len(formVals)))
				for i := 0; i < len(formVals); i++ {
					fieldV.Index(i).SetString(formVals[i])
				}
			}
		}
	}
	return nil
}

func ParseForm(form url.Values, obj interface{}) error {
	objT := reflect.TypeOf(obj)
	objV := reflect.ValueOf(obj)
	if !isStructPtr(objT) {
		return fmt.Errorf("%v must be  a struct pointer", obj)
	}
	objT = objT.Elem()
	objV = objV.Elem()

	return parseFormToStruct(form, objT, objV)
}

func GetCSVHeader(obj interface{}) ([]string, error) {
	var header []string
	header = nil
	objT := reflect.TypeOf(obj)
	objV := reflect.ValueOf(obj)
	if !isStructPtr(objT) {
		return nil, fmt.Errorf("%v must be  a struct pointer", obj)
	}
	objT = objT.Elem()
	objV = objV.Elem()
	for i := 0; i < objT.NumField(); i++ {
		fieldV := objV.Field(i)
		fieldT := objT.Field(i)
		if fieldT.Type.Kind() == reflect.Struct {
			if fieldT.Type.String() != "time.Time" {
				hdr, err := GetCSVHeader(fieldV.Addr().Interface())
				if err != nil {
					//return nil, err  //исправлено проверить в функции для получения значений
					continue
				}
				for _, name := range hdr {
					header = append(header, name)
				}
				continue
			}
		} else if fieldT.Type.Kind() == reflect.Pointer {
			zeroInt := fieldV.Pointer()
			if zeroInt != 0 {
				hdr, err := GetCSVHeader(fieldV.Interface())
				if err != nil {
					//return nil, err //исправлено
					continue
				}
				for _, name := range hdr {
					header = append(header, name)
				}
			}
			continue
		} else if fieldT.Type.Kind() == reflect.Slice {
			continue
		}

		tags := strings.Split(fieldT.Tag.Get("csv"), ",")
		var tag string
		if len(tags) == 0 || len(tags[0]) == 0 {
			tag = fieldT.Name
		} else if tags[0] == "-" {
			continue
		} else {
			tag = tags[0]
		}
		header = append(header, tag)
	}
	return header, nil
}

func GetCSVData(obj interface{}) ([]string, error) {
	var data []string
	data = nil
	objT := reflect.TypeOf(obj)
	objV := reflect.ValueOf(obj)
	if !isStructPtr(objT) {
		return nil, fmt.Errorf("%v must be a struct pointer", obj)
	}
	objT = objT.Elem()
	objV = objV.Elem()
	for i := 0; i < objT.NumField(); i++ {
		dataStr := ""
		format := ""
		fieldV := objV.Field(i)
		fieldT := objT.Field(i)
		formStr := strings.Split(fieldT.Tag.Get("format"), ",")
		if len(formStr) > 0 {
			if formStr[0] != "" {
				format = formStr[0]
			}
		}
		if fieldT.Type.Kind() == reflect.Struct {
			if fieldT.Type.String() != "time.Time" {
				d, err := GetCSVData(fieldV.Addr().Interface())
				if err != nil {
					return nil, err
				}
				for _, valStr := range d {
					data = append(data, valStr)
				}
				continue
			} else { //преобразование даты
				t := fieldV.Addr().Interface().(*time.Time)
				if format != "" {
					dataStr = t.Format(format)
				} else {
					dataStr = t.Format(formatDateTimeRus)
				}
			}
		} else if fieldT.Type.Kind() == reflect.Pointer {
			zeroInt := fieldV.Pointer()
			if zeroInt != 0 {
				d, err := GetCSVData(fieldV.Interface())
				if err != nil {
					//return nil, err //исправлено
					continue
				}
				for _, valStr := range d {
					data = append(data, valStr)
				}
			}
			continue
		} else if fieldT.Type.Kind() == reflect.Slice {
			continue
		}
		//-------------------------------------------
		switch fieldT.Type.Kind() {
		case reflect.Bool:
			if format != "" {
				dataStr = fmt.Sprintf(format, fieldV.Bool())
			} else {
				dataStr = strconv.FormatBool(fieldV.Bool())
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if format != "" {
				dataStr = fmt.Sprintf(format, fieldV.Int())
			} else {
				dataStr = strconv.FormatInt(fieldV.Int(), 10)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if format != "" {
				dataStr = fmt.Sprintf(format, fieldV.Uint())
			} else {
				dataStr = strconv.FormatUint(fieldV.Uint(), 10)
			}
		case reflect.Float32, reflect.Float64:
			f := fieldV.Float()
			if format != "" {
				if f == 0.0 {
					dataStr = "0"
				} else {
					dataStr = fmt.Sprintf(format, f)
				}
				dataStr = strings.Replace(dataStr, ".", ",", -1) //заменить символ . на ,
			} else {
				dataStr = strconv.FormatFloat(f, 'f', 6, 64)
				dataStr = strings.Replace(dataStr, ".", ",", -1) //заменить символ . на ,
			}
		case reflect.Interface:
			if format != "" {
				dataStr = fmt.Sprintf(format, fieldV.Interface())
			}
			//
		case reflect.String:
			dataStr = fieldV.String()
			s := strings.Split(dataStr, ",")
			dataStr = ""
			if len(s) > 1 {
				_ = s
			}
			for _, s1 := range s {
				dataStr += s1 + " "
			}
			s = strings.Split(dataStr, "\r")
			dataStr = ""
			if len(s) > 1 {
				_ = s
			}
			for _, s1 := range s {
				dataStr += s1
			}
			s = strings.Split(dataStr, "\n")
			dataStr = ""
			if len(s) > 1 {
				_ = s
			}
			for _, s1 := range s {
				dataStr += s1
			}

			s = strings.Split(dataStr, "\t")
			dataStr = ""
			if len(s) > 1 {
				_ = s
			}
			for _, s1 := range s {
				dataStr += s1
			}
		case reflect.Slice:
			//
		}
		//-----------------------------------------------------------------------------
		tags := strings.Split(fieldT.Tag.Get("csv"), ",")
		if len(tags) > 0 || len(tags[0]) > 0 {
			if tags[0] == "-" {
				continue
			}
		}
		data = append(data, strings.TrimSpace(dataStr))
	}
	return data, nil
}

/*
type DemoData struct {
	Id    int64 `csv:"Идентификатор2" format:"%d"`
	Value int64 `csv:"Значение" format:"%d"`
}

type FieldsCSV struct {
	Id   int64  `csv:"Идентификатор"`
	Name string `csv:"Наименование"`
	Add  DemoData
	Date time.Time `csv:"Дата" format:"2006.01.02"`
}

func main() {
	var myRec FieldsCSV
	var form url.Values
	form = make(url.Values)
	myRec.Id = 999
	myRec.Name = "Тетовое поле"
	myRec.Add.Id = 123
	myRec.Add.Value = 321
	myRec.Date = time.Now().Local()

	s1 := []string{"333"}
	s2 := []string{"Поле 1"}
	s3 := []string{"2023/05/06"}
	form["Идентификатор"] = s1
	form["Наименование"] = s2
	form["Дата"] = s3
	ParseForm(form, &myRec)

	csvFile, err := os.Create("employee.csv")
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	csvwriter := csv.NewWriter(csvFile)
	header, _ := GetCSVHeader(&myRec)
	_ = csvwriter.Write(header)
	data, _ := GetCSVData(&myRec)
	_ = csvwriter.Write(data)
	csvwriter.Flush()
	csvFile.Close()
}
*/
