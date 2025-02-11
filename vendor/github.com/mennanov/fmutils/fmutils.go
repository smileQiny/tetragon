package fmutils

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Filter keeps the msg fields that are listed in the paths and clears all the rest.
//
// This is a handy wrapper for NestedMask.Filter method.
// If the same paths are used to process multiple proto messages use NestedMask.Filter method directly.
func Filter(msg proto.Message, paths []string) {
	NestedMaskFromPaths(paths).Filter(msg)
}

// Prune clears all the fields listed in paths from the given msg.
//
// This is a handy wrapper for NestedMask.Prune method.
// If the same paths are used to process multiple proto messages use NestedMask.Filter method directly.
func Prune(msg proto.Message, paths []string) {
	NestedMaskFromPaths(paths).Prune(msg)
}

// Overwrite overwrites all the fields listed in paths in the dest msg using values from src msg.
//
// This is a handy wrapper for NestedMask.Overwrite method.
// If the same paths are used to process multiple proto messages use NestedMask.Overwrite method directly.
func Overwrite(src, dest proto.Message, paths []string) {
	NestedMaskFromPaths(paths).Overwrite(src, dest)
}

// NestedMask represents a field mask as a recursive map.
type NestedMask map[string]NestedMask

// NestedMaskFromPaths creates an instance of NestedMask for the given paths.
func NestedMaskFromPaths(paths []string) NestedMask {
	mask := make(NestedMask)
	for _, path := range paths {
		curr := mask
		var letters []rune
		for _, letter := range path {
			if letter == '.' {
				if len(letters) == 0 {
					continue
				}

				key := string(letters)
				c, ok := curr[key]
				if !ok {
					c = make(NestedMask)
					curr[key] = c
				}
				curr = c
				letters = nil
				continue
			}
			letters = append(letters, letter)
		}
		if len(letters) != 0 {
			key := string(letters)
			if _, ok := curr[key]; !ok {
				curr[key] = make(NestedMask)
			}
		}
	}

	return mask
}

// Filter keeps the msg fields that are listed in the paths and clears all the rest.
//
// If the mask is empty then all the fields are kept.
// Paths are assumed to be valid and normalized otherwise the function may panic.
// See google.golang.org/protobuf/types/known/fieldmaskpb for details.
func (mask NestedMask) Filter(msg proto.Message) {
	if len(mask) == 0 {
		return
	}

	rft := msg.ProtoReflect()
	rft.Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		m, ok := mask[string(fd.Name())]
		if ok {
			if len(m) == 0 {
				return true
			}

			if fd.IsMap() {
				xmap := rft.Get(fd).Map()
				xmap.Range(func(mk protoreflect.MapKey, mv protoreflect.Value) bool {
					if mi, ok := m[mk.String()]; ok {
						if i, ok := mv.Interface().(protoreflect.Message); ok && len(mi) > 0 {
							mi.Filter(i.Interface())
						}
					} else {
						xmap.Clear(mk)
					}

					return true
				})
			} else if fd.IsList() {
				list := rft.Get(fd).List()
				for i := 0; i < list.Len(); i++ {
					m.Filter(list.Get(i).Message().Interface())
				}
			} else if fd.Kind() == protoreflect.MessageKind {
				m.Filter(rft.Get(fd).Message().Interface())
			}
		} else {
			rft.Clear(fd)
		}
		return true
	})
}

// Prune clears all the fields listed in paths from the given msg.
//
// All other fields are kept untouched. If the mask is empty no fields are cleared.
// This operation is the opposite of NestedMask.Filter.
// Paths are assumed to be valid and normalized otherwise the function may panic.
// See google.golang.org/protobuf/types/known/fieldmaskpb for details.
func (mask NestedMask) Prune(msg proto.Message) {
	if len(mask) == 0 {
		return
	}

	rft := msg.ProtoReflect()
	rft.Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		m, ok := mask[string(fd.Name())]
		if ok {
			if len(m) == 0 {
				rft.Clear(fd)
				return true
			}

			if fd.IsMap() {
				xmap := rft.Get(fd).Map()
				xmap.Range(func(mk protoreflect.MapKey, mv protoreflect.Value) bool {
					if mi, ok := m[mk.String()]; ok {
						if i, ok := mv.Interface().(protoreflect.Message); ok && len(mi) > 0 {
							mi.Prune(i.Interface())
						} else {
							xmap.Clear(mk)
						}
					}

					return true
				})
			} else if fd.IsList() {
				list := rft.Get(fd).List()
				for i := 0; i < list.Len(); i++ {
					m.Prune(list.Get(i).Message().Interface())
				}
			} else if fd.Kind() == protoreflect.MessageKind {
				m.Prune(rft.Get(fd).Message().Interface())
			}
		}
		return true
	})
}

// Overwrite overwrites all the fields listed in paths in the dest msg using values from src msg.
//
// All other fields are kept untouched. If the mask is empty, no fields are overwritten.
// Supports scalars, messages, repeated fields, and maps.
// If the parent of the field is nil message, the parent is initiated before overwriting the field
// If the field in src is empty value, the field in dest is cleared.
// Paths are assumed to be valid and normalized otherwise the function may panic.
func (mask NestedMask) Overwrite(src, dest proto.Message) {
	mask.overwrite(src.ProtoReflect(), dest.ProtoReflect())
}

func (mask NestedMask) overwrite(src, dest protoreflect.Message) {
	for k, v := range mask {
		srcFD := src.Descriptor().Fields().ByName(protoreflect.Name(k))
		destFD := dest.Descriptor().Fields().ByName(protoreflect.Name(k))
		if srcFD == nil || destFD == nil {
			continue
		}

		// Leaf mask -> copy value from src to dest
		if len(v) == 0 {
			if srcFD.Kind() == destFD.Kind() { // TODO: Full type equality check
				val := src.Get(srcFD)
				if isValid(srcFD, val) {
					dest.Set(destFD, val)
				} else {
					dest.Clear(destFD)
				}
			}
		} else if srcFD.Kind() == protoreflect.MessageKind {
			// If dest field is nil
			if !dest.Get(destFD).Message().IsValid() {
				dest.Set(destFD, protoreflect.ValueOf(dest.Get(destFD).Message().New()))
			}
			v.overwrite(src.Get(srcFD).Message(), dest.Get(destFD).Message())
		}
	}
}

func isValid(fd protoreflect.FieldDescriptor, val protoreflect.Value) bool {
	if fd.IsMap() {
		return val.Map().IsValid()
	} else if fd.IsList() {
		return val.List().IsValid()
	} else if fd.Message() != nil {
		return val.Message().IsValid()
	}
	return true
}
