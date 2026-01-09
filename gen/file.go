package gen

import (
	"sort"
	"strconv"
)

func generateFile(packageName string, importsToGen [][]string) []byte {
	var genContent []byte

	genContent = append(genContent, []byte("// "+disclaimer+"\n\n")...)
	genContent = append(genContent, []byte("package "+packageName+"\n\n")...)

	imports := make([][]string, 0, len(importsToGen))
	imports = append(imports, importsToGen...)

	if len(imports) > 0 {
		sort.Slice(imports, func(i, j int) bool {
			return imports[i][1] < imports[j][1]
		})

		genContent = append(genContent, []byte("import (\n")...)

		for _, imp := range imports {
			quoted := strconv.Quote(imp[1])

			genContent = append(genContent, '\t')

			if imp[0] != "" {
				genContent = append(genContent, []byte(imp[0]+" "+quoted)...)
			} else {
				genContent = append(genContent, []byte(quoted)...)
			}

			genContent = append(genContent, '\n')
		}

		genContent = append(genContent, []byte(")")...)
	}

	return genContent
}
