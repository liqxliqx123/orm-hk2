package orm

import (
	"context"
	"fmt"
	"gitee.com/geektime-geekbang/geektime-go/orm/internal/errs"
	"gitee.com/geektime-geekbang/geektime-go/orm/model"
	"log"
	"reflect"
	"strings"
	"unicode"
)

type Updater[T any] struct {
	builder
	db      *DB
	assigns []Assignable
	val     *T
	where   []Predicate
}

func NewUpdater[T any](db *DB) *Updater[T] {
	register, err := model.NewRegistry().Register(new(T))
	if err != nil {
		log.Fatalln(err)
		return nil
	}
	return &Updater[T]{
		builder: builder{
			sb:      strings.Builder{},
			args:    nil,
			model:   register,
			dialect: nil,
			quoter:  '`',
		},
		db:      db,
		assigns: make([]Assignable, 0),
		val:     new(T),
		where:   make([]Predicate, 0),
	}
}

func (u *Updater[T]) Update(t *T) *Updater[T] {
	u.val = t
	return u
}

func (u *Updater[T]) Set(assigns ...Assignable) *Updater[T] {
	u.assigns = assigns
	return u
}

func (u *Updater[T]) Build() (*Query, error) {
	if len(u.assigns) == 0 {
		return nil, errs.ErrNoUpdatedColumns
	}
	if u.val == nil {
		u.val = new(T)
	}

	u.sb.WriteString("UPDATE ")
	u.quote(u.model.TableName)
	u.sb.WriteString(" SET ")

	for i, a := range u.assigns {
		if i > 0 {
			u.sb.WriteByte(',')
		}
		switch assign := a.(type) {
		case Column:
			if err := u.buildColumn(assign.name); err != nil {
				return nil, err
			}
			u.sb.WriteString("=?")
			of := reflect.ValueOf(u.val)
			arg := of.Elem().FieldByName(assign.name)
			u.addArgs(arg.Interface())
		case Assignment:
			u.quote(underscoreName(assign.column))

			var v any
			switch vt := assign.val.(type) {
			case value:
				v = vt.val
			case MathExpr:
				v2, ok := vt.right.(value)
				if ok {
					v = v2.val
				}
				column, ok := vt.left.(Column)
				if ok {
					u.sb.WriteString("=")

					u.sb.WriteString(fmt.Sprintf("`%s`", underscoreName(column.name)))
					u.sb.WriteString(fmt.Sprintf(" %v ", vt.op))
					u.sb.WriteString("?")
				}

			}

			u.addArgs(v)
		default:
			return nil, errs.NewErrUnsupportedAssignableType(a)
		}
	}
	if len(u.where) > 0 {
		u.sb.WriteString(" WHERE ")
		if err := u.buildPredicates(u.where); err != nil {
			return nil, err
		}
	}
	u.sb.WriteByte(';')
	return &Query{
		SQL:  u.sb.String(),
		Args: u.args,
	}, nil
	return nil, nil
}

func (u *Updater[T]) Where(ps ...Predicate) *Updater[T] {
	u.where = ps
	return u
}

func (u *Updater[T]) Exec(ctx context.Context) Result {
	panic("implement me")
}

// AssignNotZeroColumns 更新非零值
func AssignNotZeroColumns(entity interface{}) []Assignable {
	ass := make([]Assignable, 0)
	return ass
}

func underscoreName(tableName string) string {
	var buf []byte
	for i, v := range tableName {
		if unicode.IsUpper(v) {
			if i != 0 {
				buf = append(buf, '_')
			}
			buf = append(buf, byte(unicode.ToLower(v)))
		} else {
			buf = append(buf, byte(v))
		}

	}
	return string(buf)
}
