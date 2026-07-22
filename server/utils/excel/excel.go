// Package excel 基于 excelize 的轻量导入导出工具。
//
// 设计目标:贴合 devops-admin 现有分层(模块专用 handler),不引入 gva sys_export_template 那套
// 动态模板/JOIN/自定义 SQL(对固定业务模块属过度设计)。调用方只需定义列(字段名→中文表头),
// 传入结构体切片即可导出;导入按表头映射反解为 map 供 service 落库。
//
// 参考 gva sys_export_template 的两点做法:① excelize 类型化的 SetCellValue
// (数字存数字、时间格式化);② 导入表头↔字段映射解析。
package excel

import (
	"bytes"
	"errors"
	"fmt"
	"mime/multipart"
	"reflect"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Header 导出列定义:Field 为 struct 字段名(支持嵌入基座的提升字段,如 CreatedAt),
// Title 为 Excel 表头(中文)。Field 走 reflect.FieldByName,自动穿透匿名嵌入字段。
type Header struct {
	Field string // 结构体字段名(如 "UserName"、"CreatedAt")
	Title string // Excel 表头文案
}

// defaultSheet excelize.NewFile 默认创建的工作表名;导出时重命名为业务名,导入时读第一个工作表。
const defaultSheet = "Sheet1"

// Export 把结构体切片 rows 按 headers 导出为 xlsx 字节缓冲。
// rows 必须是切片(或切片指针);每个元素的指定字段经反射读取,数字存为数字单元格,
// time.Time 格式化为 "2006-01-02 15:04:05",nil 指针留空。
// 注意:用 SetSheetName 重命名默认工作表,保持单 sheet(多 sheet 会致导入读到空工作表)。
func Export(rows any, headers []Header, sheetName string) (*bytes.Buffer, error) {
	if len(headers) == 0 {
		return nil, errors.New("导出列定义为空")
	}
	if sheetName == "" {
		sheetName = defaultSheet
	}
	f := excelize.NewFile()
	defer f.Close()
	if sheetName != defaultSheet {
		f.SetSheetName(defaultSheet, sheetName)
	}
	// 表头
	for i, h := range headers {
		if err := f.SetCellValue(sheetName, axis(i+1, 1), h.Title); err != nil {
			return nil, err
		}
	}
	// 数据行
	rv := reflect.ValueOf(rows)
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice {
		return nil, errors.New("rows 必须是切片")
	}
	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i)
		for item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		for j, h := range headers {
			cell := axis(j+1, i+2)
			if err := f.SetCellValue(sheetName, cell, cellValue(item, h.Field)); err != nil {
				return nil, err
			}
		}
	}
	return f.WriteToBuffer()
}

// ExportTemplate 只写表头(供导入下载空模板)。同样保持单 sheet。
func ExportTemplate(headers []Header, sheetName string) (*bytes.Buffer, error) {
	if len(headers) == 0 {
		return nil, errors.New("导出列定义为空")
	}
	if sheetName == "" {
		sheetName = defaultSheet
	}
	f := excelize.NewFile()
	defer f.Close()
	if sheetName != defaultSheet {
		f.SetSheetName(defaultSheet, sheetName)
	}
	for i, h := range headers {
		if err := f.SetCellValue(sheetName, axis(i+1, 1), h.Title); err != nil {
			return nil, err
		}
	}
	return f.WriteToBuffer()
}

// Parse 读取上传的 xlsx,按 headers 的 Title 匹配首行表头,反解为记录切片。
// 每条记录为 map[字段名]单元格字符串;模板中未定义的多余列被忽略,记录中缺失的字段不出现。
// 全空行跳过。要求首行必须是表头(与 ExportTemplate 产物一致)。
// 读取第一个工作表,以适配任意 sheet 名(模板下载的 sheet 为业务名,用户自建 Excel 可能是 Sheet1/工作表1)。
func Parse(fileHeader *multipart.FileHeader, headers []Header) ([]map[string]string, error) {
	if fileHeader == nil {
		return nil, errors.New("文件为空")
	}
	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("打开上传文件失败: %w", err)
	}
	defer src.Close()
	f, err := excelize.OpenReader(src)
	if err != nil {
		return nil, fmt.Errorf("解析 Excel 失败: %w", err)
	}
	defer f.Close()
	// 读第一个工作表,适配用户上传的任意 sheet 名
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("Excel 无有效工作表")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("读取 Excel 行失败: %w", err)
	}
	if len(rows) < 2 {
		return nil, errors.New("Excel 无有效数据,至少应包含表头行与一行数据")
	}
	// 表头标题 → 字段名
	titleField := make(map[string]string, len(headers))
	for _, h := range headers {
		titleField[strings.TrimSpace(h.Title)] = h.Field
	}
	headerRow := rows[0]
	// 记录每列表头对应的字段名(无映射的列记空,跳过)
	colFields := make([]string, len(headerRow))
	for i, title := range headerRow {
		colFields[i] = titleField[strings.TrimSpace(title)]
	}
	result := make([]map[string]string, 0, len(rows)-1)
	for _, row := range rows[1:] {
		item := make(map[string]string, len(headers))
		nonEmpty := false
		for i, val := range row {
			if i >= len(colFields) || colFields[i] == "" {
				continue
			}
			v := strings.TrimSpace(val)
			if v != "" {
				nonEmpty = true
			}
			item[colFields[i]] = v
		}
		if nonEmpty {
			result = append(result, item)
		}
	}
	return result, nil
}

// cellValue 反射读取结构体 item 的指定字段并转为 excelize 可识别的单元格值。
// 穿透匿名嵌入字段(reflect.FieldByName 自带);指针解引用,nil 指针返回 ""。
func cellValue(item reflect.Value, field string) any {
	fv := item.FieldByName(field)
	if !fv.IsValid() {
		return ""
	}
	for fv.Kind() == reflect.Ptr || fv.Kind() == reflect.Interface {
		if fv.IsNil() {
			return ""
		}
		fv = fv.Elem()
	}
	if t, ok := fv.Interface().(time.Time); ok {
		if t.IsZero() {
			return ""
		}
		return t.Format("2006-01-02 15:04:05")
	}
	switch fv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fv.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(fv.Uint())
	case reflect.Float32, reflect.Float64:
		return fv.Float()
	case reflect.Bool:
		return fv.Bool()
	default:
		return fmt.Sprintf("%v", fv.Interface())
	}
}

// axis 把(列号, 行号)转为 Excel 单元元地址,如 (1,1)→"A1"、(27,2)→"AA2"。
func axis(col, row int) string {
	return colName(col) + fmt.Sprintf("%d", row)
}

// colName 列号(从 1 开始)转 Excel 列字母。
func colName(n int) string {
	name := ""
	for n > 0 {
		n--
		name = string(rune('A'+n%26)) + name
		n /= 26
	}
	return name
}
