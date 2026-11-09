// Package convaudit - вспомогательные функции для тестов конвертеров
// "доменная модель <-> gRPC-сообщение" (grpcserver/message_*.go,
// grpcclient/message_*.go). Идея: заполнить исходную структуру ненулевыми
// значениями во ВСЕХ экспортируемых полях (Fill), выполнить конвертацию и
// найти поля результата, оставшиеся нулевыми (ZeroFields) - это либо забытое
// при копировании поле, либо поле без аналога в источнике (допустимые случаи
// перечисляются в allow-списках конкретных тестов).
package convaudit

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"
)

const maxDepth = 5

var fillTime = time.Date(2026, 1, 2, 3, 4, 5, 123456789, time.UTC)

// Fill заполняет ptr (указатель на структуру) ненулевыми значениями. Срезы
// получают ровно один элемент, карты - одну запись, указатели выделяются.
// Неэкспортируемые поля пропускаются. Глубина рекурсии ограничена (maxDepth),
// чтобы не зациклиться на ссылающихся друг на друга структурах.
func Fill(ptr any) {
	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		panic("convaudit.Fill: нужен непустой указатель")
	}
	fill(v.Elem(), 0, 1)
}

var counter float64

func fill(v reflect.Value, depth int, seed int) {
	if depth > maxDepth || !v.CanSet() {
		return
	}
	switch v.Kind() {
	case reflect.String:
		v.SetString("s")
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(int64(seed%100) + 2)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(uint64(seed%100) + 2)
	case reflect.Float32, reflect.Float64:
		counter++
		v.SetFloat(counter + 0.5)
	case reflect.Ptr:
		n := reflect.New(v.Type().Elem())
		fill(n.Elem(), depth+1, seed)
		v.Set(n)
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 1, 1)
		fill(s.Index(0), depth+1, seed)
		v.Set(s)
	case reflect.Map:
		m := reflect.MakeMap(v.Type())
		k := reflect.New(v.Type().Key()).Elem()
		fill(k, depth+1, seed)
		e := reflect.New(v.Type().Elem()).Elem()
		fill(e, depth+1, seed)
		m.SetMapIndex(k, e)
		v.Set(m)
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			v.Set(reflect.ValueOf(fillTime))
			return
		}
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if !unicode.IsUpper([]rune(f.Name)[0]) {
				continue
			}
			fill(v.Field(i), depth, seed+i)
		}
	}
}

// ZeroFields возвращает отсортированный список путей экспортируемых полей
// структуры ptr, оставшихся нулевыми (пустая строка, 0, false, nil, пустой
// срез/карта, нулевое время). Рекурсивно заходит в вложенные структуры,
// указатели и первый элемент срезов.
func ZeroFields(ptr any) []string {
	v := reflect.ValueOf(ptr)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	var res []string
	walkZero(v, "", 0, &res)
	sort.Strings(res)
	return res
}

func walkZero(v reflect.Value, path string, depth int, res *[]string) {
	if depth > maxDepth {
		return
	}
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			*res = append(*res, path)
			return
		}
		walkZero(v.Elem(), path, depth+1, res)
	case reflect.Slice:
		if v.Len() == 0 {
			*res = append(*res, path)
			return
		}
		walkZero(v.Index(0), path+"[]", depth+1, res)
	case reflect.Map:
		if v.Len() == 0 {
			*res = append(*res, path)
		}
	case reflect.Struct:
		if v.Type() == reflect.TypeOf(time.Time{}) {
			if v.Interface().(time.Time).IsZero() {
				*res = append(*res, path)
			}
			return
		}
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if !unicode.IsUpper([]rune(f.Name)[0]) {
				continue
			}
			p := f.Name
			if path != "" {
				p = path + "." + f.Name
			}
			walkZero(v.Field(i), p, depth, res)
		}
	default:
		if v.IsZero() {
			*res = append(*res, path)
		}
	}
}

// Unexpected возвращает элементы got, которых нет в allow. Элемент allow
// совпадает с got по точному пути либо по префиксу "путь." / "путь[]".
func Unexpected(got, allow []string) []string {
	var res []string
	for _, g := range got {
		ok := false
		for _, a := range allow {
			if g == a || strings.HasPrefix(g, a+".") || strings.HasPrefix(g, a+"[]") {
				ok = true
				break
			}
		}
		if !ok {
			res = append(res, g)
		}
	}
	return res
}

// Report форматирует результат аудита одного конвертера.
func Report(name string, fields []string) string {
	return fmt.Sprintf("%s: проблемные поля (%d): %s", name, len(fields), strings.Join(fields, ", "))
}

// Diff сравнивает две структуры одного типа (указатели на них) по
// экспортируемым полям и возвращает отсортированные пути отличающихся
// листовых полей. Используется для round-trip проверки "домен -> gRPC ->
// домен": отличия - поля, потерянные одним из двух конвертеров либо не
// имеющие поля в сообщении (пробел в контракте protocol/*.proto).
func Diff(a, b any) []string {
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	if va.Kind() == reflect.Ptr {
		va, vb = va.Elem(), vb.Elem()
	}
	var res []string
	walkDiff(va, vb, "", 0, &res)
	sort.Strings(res)
	return res
}

func walkDiff(a, b reflect.Value, path string, depth int, res *[]string) {
	if depth > maxDepth {
		return
	}
	switch a.Kind() {
	case reflect.Ptr:
		if a.IsNil() != b.IsNil() {
			*res = append(*res, path)
			return
		}
		if !a.IsNil() {
			walkDiff(a.Elem(), b.Elem(), path, depth+1, res)
		}
	case reflect.Slice:
		if a.Len() != b.Len() {
			*res = append(*res, path)
			return
		}
		for i := 0; i < a.Len(); i++ {
			walkDiff(a.Index(i), b.Index(i), path+"[]", depth+1, res)
		}
	case reflect.Map:
		if a.Len() != b.Len() {
			*res = append(*res, path)
		}
	case reflect.Struct:
		if a.Type() == reflect.TypeOf(time.Time{}) {
			if !a.Interface().(time.Time).Equal(b.Interface().(time.Time)) {
				*res = append(*res, path)
			}
			return
		}
		for i := 0; i < a.NumField(); i++ {
			f := a.Type().Field(i)
			if !unicode.IsUpper([]rune(f.Name)[0]) {
				continue
			}
			p := f.Name
			if path != "" {
				p = path + "." + f.Name
			}
			walkDiff(a.Field(i), b.Field(i), p, depth, res)
		}
	default:
		if !reflect.DeepEqual(a.Interface(), b.Interface()) {
			*res = append(*res, path)
		}
	}
}
