package build

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Template TODO
type Template struct {
	Files   []*File        `json:"files"`
	Scalars []*ScalarValue `json:"scalarValueTypes"`
}

// ParseFiles TODO
func (tmpl *Template) ParseFiles(filenames ...string) error {
	for _, filename := range filenames {
		newer, err := new(Template).doParseFile(filename)
		if err != nil {
			return err
		}

		for _, file := range newer.Files {
			err = tmpl.appendFile(file)
			if err != nil {
				return err
			}
		}

		for _, scalar := range newer.Scalars {
			tmpl.appendScalarValue(scalar)
		}
	}

	tmpl.finishParse()
	tmpl.buildMessagesJsonString()
	return nil
}

// Object TODO
type Object struct {
	*Enum
	*Message
	*ScalarValue
	JSONObject interface{}
}

func (tmpl *Template) buildMessagesJsonString() {
	objects := map[string]Object{
		"google.protobuf.BoolValue":   {JSONObject: "bool"},
		"google.protobuf.Int32Value":  {JSONObject: "int32"},
		"google.protobuf.Int64Value":  {JSONObject: "int64"},
		"google.protobuf.UInt32Value": {JSONObject: "uint32"},
		"google.protobuf.UInt64Value": {JSONObject: "uint64"},
		"google.protobuf.FloatValue":  {JSONObject: "float"},
		"google.protobuf.DoubleValue": {JSONObject: "double"},
		"google.protobuf.StringValue": {JSONObject: "string"},
		"google.protobuf.BytesValue":  {JSONObject: "bytes"},
		"google.protobuf.Duration": {JSONObject: map[string]interface{}{
			"seconds": "int64",
			"nanos":   "int32",
		}},
		"google.protobuf.Timestamp": {JSONObject: map[string]interface{}{
			"seconds": "int64",
			"nanos":   "int32",
		}},
		"google.protobuf.Value": {JSONObject: "any json object or any json array, same as interface{}"},
		"google.protobuf.ListValue": {JSONObject: []interface{}{
			"any json object or any json array, same as []interface{}",
		}},
		"google.protobuf.Struct": {JSONObject: map[string]interface{}{
			"string": "any json object or any json array, same as map[string]interface{}",
		}},
	}

	for _, file := range tmpl.Files {
		for _, enum := range file.Enums {
			enum.File = file
			objects[enum.FullName] = Object{Enum: enum}
		}

		for _, message := range file.Messages {
			message.File = file
			objects[message.FullName] = Object{Message: message}
		}
	}

	for _, scalar := range tmpl.Scalars {
		objects[scalar.ProtoType] = Object{ScalarValue: scalar}
	}

	for _, file := range tmpl.Files {
		for _, message := range file.Messages {
			o, err := tmpl.fromMessage(objects, message)
			if err != nil {
				log.Printf("encode msg to json failure, error: %v", err)
				continue
			}

			message.JSONObject = o.(map[string]interface{})
		}
	}
}

func (tmpl *Template) fromEnum(_ *Enum) (interface{}, error) {
	return "int64", nil
}

func (tmpl *Template) fromScalarValue(value *ScalarValue) (interface{}, error) {
	switch value.ProtoType {
	case "float", "double":
		fallthrough
	case "int32", "int64", "sint32", "sint64", "sfixed32", "sfixed64":
		fallthrough
	case "uint32", "uint64":
		fallthrough
	case "bool":
		fallthrough
	case "string", "bytes":
		return value.ProtoType, nil
	default:
		return nil, fmt.Errorf("unknown scalar type: %s", value.ProtoType)
	}
}

func (tmpl *Template) fromMessage(objects map[string]Object, value *Message) (interface{}, error) {
	res := make(map[string]interface{})

	for _, f := range value.Fields {
		v, err := tmpl.fromObjectName(objects, f.FullType)
		if err != nil {
			return nil, err
		}

		switch {
		case f.Isarray:
			res[f.Name] = []interface{}{v}

		case f.Ismap:
			_, err := tmpl.fromObjectName(objects, f.KeyFullType)
			if err != nil {
				return nil, fmt.Errorf("map key except scalar type, but %s", f.KeyFullType)
			}

			res[f.Name] = map[string]interface{}{"string": v}
		default:
			res[f.Name] = v
		}
	}
	return res, nil
}

func (tmpl *Template) fromObjectName(objects map[string]Object, objectName string) (interface{}, error) {
	var err error

	obj, ok := objects[objectName]
	if !ok {
		return nil, fmt.Errorf("unknown object: %s", objectName)
	}

	if obj.JSONObject != nil {
		return obj.JSONObject, nil
	}

	switch {
	case obj.Enum != nil:
		obj.JSONObject, err = tmpl.fromEnum(obj.Enum)
	case obj.Message != nil:
		obj.JSONObject, err = tmpl.fromMessage(objects, obj.Message)
	case obj.ScalarValue != nil:
		obj.JSONObject, err = tmpl.fromScalarValue(obj.ScalarValue)
	default:
		return nil, fmt.Errorf("unknown error")
	}

	return obj.JSONObject, err
}

func (tmpl *Template) doParseFile(filename string) (*Template, error) {
	fp, err := os.Open(filename)
	if err != nil {
		return tmpl, fmt.Errorf("open file failure, path: %s, err: %w", filename, err)
	}

	defer func() {
		_ = fp.Close()
	}()

	dec := json.NewDecoder(fp)
	err = dec.Decode(&tmpl)
	if err != nil {
		return tmpl, fmt.Errorf("decode file failure, path: %s, err: %w", filename, err)
	}

	return tmpl, nil
}

func (tmpl *Template) appendFile(newer *File) error {
	if strings.HasSuffix(newer.Name, "validate/validate.proto") {
		return nil
	}

	var file *File

	newer.Dir = filepath.Clean(filepath.Dir(newer.Name))

	for _, iter := range tmpl.Files {
		if iter.Dir == newer.Dir {
			file = iter
			break
		}
	}

	if file == nil {
		tmpl.Files = append(tmpl.Files, newer)
		return nil
	}

	if file.Package != newer.Package {
		return fmt.Errorf("%s and %s has different package", file.Name, newer.Name)
	}

	for _, enum := range newer.Enums {
		file.Enums = append(file.Enums, enum)
	}

	for _, extension := range newer.Extensions {
		file.Extensions = append(file.Extensions, extension)
	}

	for _, message := range newer.Messages {
		file.Messages = append(file.Messages, message)
	}

	for _, service := range newer.Services {
		service.File = file
		file.Services = append(file.Services, service)
	}

	file.HasEnums = len(file.Enums) > 0
	file.HasExtensions = len(file.Extensions) > 0
	file.HasMessages = len(file.Messages) > 0
	file.HasServices = len(file.Services) > 0
	file.Description = ""
	file.Options = nil
	return nil
}

func (tmpl *Template) appendScalarValue(scalarValue *ScalarValue) {
	found := false

	for _, dst := range tmpl.Scalars {
		if scalarValue.ProtoType == dst.ProtoType {
			found = true
			break
		}
	}

	if !found {
		tmpl.Scalars = append(tmpl.Scalars, scalarValue)
	}
}

func (tmpl *Template) finishParse() {
	for _, file := range tmpl.Files {
		for _, enum := range file.Enums {
			enum.File = file
		}

		for _, extension := range file.Extensions {
			extension.File = file
		}

		for _, message := range file.Messages {
			message.File = file

			for _, field := range message.Fields {
				tmpl.handleMapField(field)
			}
		}

		for _, service := range file.Services {
			service.File = file

			for _, method := range service.Methods {
				method.Service = service
			}
		}
	}

	tmpl.sort()
}

func (tmpl *Template) handleMapField(messageField *MessageField) {

	var mapEntry *Message

	if messageField.Label != "repeated" || messageField.Done {
		return
	}

	if !messageField.Ismap {
		messageField.Label = "array"
		messageField.Isarray = true
		messageField.Ismap = false
		messageField.Done = true
		return
	}

	for _, file := range tmpl.Files {
		for _, message := range file.Messages {
			if message.FullName == messageField.FullType {
				mapEntry = message
				break
			}
		}
	}

	var key, value *MessageField

	for _, field := range mapEntry.Fields {
		switch field.Name {
		case "key":
			key = field
		case "value":
			value = field
		}
	}

	if key == nil || value == nil || len(mapEntry.Fields) != 2 {
		messageField.Label = "array"
		messageField.Isarray = true
		messageField.Ismap = false
		messageField.Done = true
		return
	}

	mapEntry.Ismapentry = true

	messageField.Label = "map"
	messageField.Isarray = false
	messageField.Ismap = true
	messageField.KeyType = key.Type
	messageField.KeyLongType = key.LongType
	messageField.KeyFullType = key.FullType

	messageField.Type = value.Type
	messageField.LongType = value.LongType
	messageField.FullType = value.FullType

	messageField.Done = true
}

func (tmpl *Template) sort() {
	for _, file := range tmpl.Files {
		sort.Slice(file.Enums, func(i, j int) bool {
			return strings.Compare(file.Enums[i].Name, file.Enums[j].Name) < 0
		})

		sort.Slice(file.Messages, func(i, j int) bool {
			return strings.Compare(file.Messages[i].Name, file.Messages[j].Name) < 0
		})

		sort.Slice(file.Services, func(i, j int) bool {
			return strings.Compare(file.Services[i].Name, file.Services[j].Name) < 0
		})

		// for _, message := range file.Messages {
		// 	sort.Slice(message.Fields, func(i, j int) bool {
		// 		return strings.Compare(message.Fields[i].Name, message.Fields[j].Name) < 0
		// 	})
		// }

		for _, service := range file.Services {
			sort.Slice(service.Methods, func(i, j int) bool {
				return strings.Compare(service.Methods[i].Name, service.Methods[j].Name) < 0
			})
		}
	}

	sort.Slice(tmpl.Files, func(i, j int) bool {
		return strings.Compare(tmpl.Files[i].Package, tmpl.Files[j].Package) < 0
	})
}

// File TODO
type File struct {
	Dir           string           `json:"-"`
	Name          string           `json:"name"`
	Description   string           `json:"description"`
	Package       string           `json:"package"`
	HasEnums      bool             `json:"hasEnums"`
	HasExtensions bool             `json:"hasExtensions"`
	HasMessages   bool             `json:"hasMessages"`
	HasServices   bool             `json:"hasServices"`
	Enums         []*Enum          `json:"enums"`
	Extensions    []*FileExtension `json:"extensions"`
	Messages      []*Message       `json:"messages"`
	Services      []*Service       `json:"services"`
	Options       Options          `json:"options,omitempty"`
}

// Option returns the named option.
func (f File) Option(name string) *ValidatorExtension {
	if f.Options != nil {
		return f.Options[name]
	} else {
		return nil
	}
}

// FileExtension contains details about top-level extensions within a proto(2) file.
type FileExtension struct {
	File               *File  `json:"-"`
	Name               string `json:"name"`
	LongName           string `json:"longName"`
	FullName           string `json:"fullName"`
	Description        string `json:"description"`
	Label              string `json:"label"`
	Type               string `json:"type"`
	LongType           string `json:"longType"`
	FullType           string `json:"fullType"`
	Number             int    `json:"number"`
	DefaultValue       string `json:"defaultValue"`
	ContainingType     string `json:"containingType"`
	ContainingLongType string `json:"containingLongType"`
	ContainingFullType string `json:"containingFullType"`
}

// Message TODO
type Message struct {
	File          *File                  `json:"-"`
	Name          string                 `json:"name"`
	LongName      string                 `json:"longName"`
	FullName      string                 `json:"fullName"`
	Description   string                 `json:"description"`
	HasExtensions bool                   `json:"hasExtensions"`
	HasFields     bool                   `json:"hasFields"`
	HasOneofs     bool                   `json:"hasOneofs"`
	Extensions    []*MessageExtension    `json:"extensions"`
	Fields        []*MessageField        `json:"fields"`
	Options       Options                `json:"options,omitempty"`
	Ismapentry    bool                   `json:"-"`
	JSONObject    map[string]interface{} `json:"-"`
}

// Option returns the named option.
func (m Message) Option(name string) *ValidatorExtension {
	if m.Options != nil {
		return m.Options[name]
	} else {
		return nil
	}
}

func (m Message) deepJSONObject(obj interface{}, deep int) interface{} {
	if deep == 0 {
		switch obj.(type) {
		case []interface{}:
			return []interface{}{}
		case map[string]interface{}:
			return map[string]interface{}{}
		default:
			return obj
		}
	}

	switch x := obj.(type) {
	case []interface{}:
		res := make([]interface{}, 0, len(x))
		for _, v := range x {
			res = append(res, m.deepJSONObject(v, deep-1))
		}
		return res
	case map[string]interface{}:
		res := make(map[string]interface{})
		for k, v := range x {
			res[k] = m.deepJSONObject(v, deep-1)
		}
		return res
	default:
		return obj
	}
}

// JSONString TODO
func (m Message) JSONString(deep int) string {
	o := m.deepJSONObject(m.JSONObject, deep)
	bs, _ := json.MarshalIndent(o, "", "  ")
	return string(bs)
}

// FieldOptions returns all options that are set on the fields in this message.
func (m Message) FieldOptions() []string {
	optionSet := make(map[string]struct{})
	for _, field := range m.Fields {
		for option := range field.Options {
			optionSet[option] = struct{}{}
		}
	}
	if len(optionSet) == 0 {
		return nil
	}
	options := make([]string, 0, len(optionSet))
	for option := range optionSet {
		options = append(options, option)
	}
	sort.Strings(options)
	return options
}

// FieldsWithOption returns all fields that have the given option set.
// If no single value has the option set, this returns nil.
func (m Message) FieldsWithOption(optionName string) []*MessageField {
	fields := make([]*MessageField, 0, len(m.Fields))
	for _, field := range m.Fields {
		if _, ok := field.Options[optionName]; ok {
			fields = append(fields, field)
		}
	}
	if len(fields) > 0 {
		return fields
	}
	return nil
}

// MessageField TODO
type MessageField struct {
	Message      *Message `json:"-"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Label        string   `json:"label"`
	Type         string   `json:"type"`
	LongType     string   `json:"longType"`
	FullType     string   `json:"fullType"`
	Ismap        bool     `json:"ismap"`
	Isoneof      bool     `json:"isoneof"`
	Oneofdecl    string   `json:"oneofdecl"`
	DefaultValue string   `json:"defaultValue"`
	Options      Options  `json:"options,omitempty"`
	Done         bool     `json:"-"`
	Isarray      bool     `json:"-"`
	KeyType      string   `json:"-"`
	KeyLongType  string   `json:"-"`
	KeyFullType  string   `json:"-"`
}

// Option returns the named option.
func (f MessageField) Option(name string) *ValidatorExtension {
	if f.Options != nil {
		return f.Options[name]
	} else {
		return nil
	}
}

// MessageExtension contains details about message-scoped extensions in proto(2) files.
type MessageExtension struct {
	FileExtension

	ScopeType     string `json:"scopeType"`
	ScopeLongType string `json:"scopeLongType"`
	ScopeFullType string `json:"scopeFullType"`
}

// Enum TODO
type Enum struct {
	File        *File        `json:"-"`
	Name        string       `json:"name"`
	LongName    string       `json:"longName"`
	FullName    string       `json:"fullName"`
	Description string       `json:"description"`
	Values      []*EnumValue `json:"values"`
	Options     Options      `json:"options,omitempty"`
}

// Option returns the named option.
func (e Enum) Option(name string) *ValidatorExtension {
	if e.Options != nil {
		return e.Options[name]
	} else {
		return nil
	}
}

// ValueOptions returns all options that are set on the values in this enum.
func (e Enum) ValueOptions() []string {
	optionSet := make(map[string]struct{})
	for _, value := range e.Values {
		for option := range value.Options {
			optionSet[option] = struct{}{}
		}
	}
	if len(optionSet) == 0 {
		return nil
	}
	options := make([]string, 0, len(optionSet))
	for option := range optionSet {
		options = append(options, option)
	}
	sort.Strings(options)
	return options
}

// ValuesWithOption returns all values that have the given option set.
// If no single value has the option set, this returns nil.
func (e Enum) ValuesWithOption(optionName string) []*EnumValue {
	values := make([]*EnumValue, 0, len(e.Values))
	for _, value := range e.Values {
		if _, ok := value.Options[optionName]; ok {
			values = append(values, value)
		}
	}
	if len(values) > 0 {
		return values
	}
	return nil
}

// EnumValue TODO
type EnumValue struct {
	Name        string  `json:"name"`
	Number      string  `json:"number"`
	Description string  `json:"description"`
	Options     Options `json:"options,omitempty"`
}

// Option returns the named option.
func (v EnumValue) Option(name string) *ValidatorExtension {
	if v.Options != nil {
		return v.Options[name]
	} else {
		return nil
	}
}

// Service TODO
type Service struct {
	File        *File            `json:"-"`
	Name        string           `json:"name"`
	LongName    string           `json:"longName"`
	FullName    string           `json:"fullName"`
	Description string           `json:"description"`
	Methods     []*ServiceMethod `json:"methods"`
	Options     Options          `json:"options,omitempty"`
}

// Option returns the named option.
func (s Service) Option(name string) *ValidatorExtension {
	if s.Options != nil {
		return s.Options[name]
	} else {
		return nil
	}
}

// MethodOptions returns all options that are set on the methods in this service.
func (s Service) MethodOptions() []string {
	optionSet := make(map[string]struct{})
	for _, method := range s.Methods {
		for option := range method.Options {
			optionSet[option] = struct{}{}
		}
	}
	if len(optionSet) == 0 {
		return nil
	}
	options := make([]string, 0, len(optionSet))
	for option := range optionSet {
		options = append(options, option)
	}
	sort.Strings(options)
	return options
}

// MethodsWithOption returns all methods that have the given option set.
// If no single method has the option set, this returns nil.
func (s Service) MethodsWithOption(optionName string) []*ServiceMethod {
	methods := make([]*ServiceMethod, 0, len(s.Methods))
	for _, method := range s.Methods {
		if _, ok := method.Options[optionName]; ok {
			methods = append(methods, method)
		}
	}
	if len(methods) > 0 {
		return methods
	}
	return nil
}

// ServiceMethod TODO
type ServiceMethod struct {
	Service           *Service `json:"-"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	RequestType       string   `json:"requestType"`
	RequestLongType   string   `json:"requestLongType"`
	RequestFullType   string   `json:"requestFullType"`
	RequestStreaming  bool     `json:"requestStreaming"`
	ResponseType      string   `json:"responseType"`
	ResponseLongType  string   `json:"responseLongType"`
	ResponseFullType  string   `json:"responseFullType"`
	ResponseStreaming bool     `json:"responseStreaming"`
	Options           Options  `json:"options,omitempty"`
}

// Option returns the named option.
func (m ServiceMethod) Option(name string) *ValidatorExtension {
	if m.Options != nil {
		return m.Options[name]
	} else {
		return nil
	}
}

// Options TODO
type Options map[string]*ValidatorExtension

// UnmarshalJSON TODO
func (o *Options) UnmarshalJSON(b []byte) error {
	var in map[string]interface{}

	err := json.Unmarshal(b, &in)
	if err != nil {
		return err
	}

	if *o == nil {
		*o = make(map[string]*ValidatorExtension)
	}

	for k, v := range in {
		switch k {
		case "validate.rules":
			var extension ValidatorExtension
			bs, _ := json.Marshal(v)
			err = json.Unmarshal(bs, &extension.rules)
			(*o)["validate.rules"] = &extension
		}
	}

	return err
}

// ValidatorRule TODO
type ValidatorRule struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// ValidatorExtension TODO
type ValidatorExtension struct {
	rules []ValidatorRule `json:"validate.rules"`
}

// Rules TODO
func (v ValidatorExtension) Rules() []ValidatorRule {
	return v.rules
}

// ScalarValue contains information about scalar value types in protobuf. The common use case for this type is to know
// which language specific type maps to the protobuf type.
//
// For example, the protobuf type `int64` maps to `long` in C#, and `Bignum` in Ruby. For the full list, take a look at
// https://developers.google.com/protocol-buffers/docs/proto3#scalar
type ScalarValue struct {
	ProtoType  string `json:"protoType"`
	Notes      string `json:"notes"`
	CppType    string `json:"cppType"`
	CSharp     string `json:"csType"`
	GoType     string `json:"goType"`
	JavaType   string `json:"javaType"`
	PhpType    string `json:"phpType"`
	PythonType string `json:"pythonType"`
	RubyType   string `json:"rubyType"`
}
