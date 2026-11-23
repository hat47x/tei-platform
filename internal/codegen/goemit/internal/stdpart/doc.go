// Package stdpart holds the standard parts that goemit copies into every
// generated package. Each part is a plain Go file that uses only the standard
// library; goemit renames its package clause and adds the generated-code
// header. Keeping the parts here lets them be compiled and tested directly.
package stdpart
