package main

import (
	"fmt"
	"github.com/kniren/gota/dataframe"
	"strings"
)

func main(){
	csvStr := `
Country,Date,Age,Amount,Id
"United States",2012-02-01,50,112.1,01234
"United States",2012-02-01,32,321.31,54320
"United Kingdom",2012-02-01,17,18.2,12345
"United States",2012-02-01,32,321.31,54320
"United Kingdom",2012-02-01,NA,18.2,12345
"United States",2012-02-01,32,321.31,54320
"United States",2012-02-01,32,321.31,54320
Spain,2012-02-01,66,555.42,00241
`
	df := dataframe.ReadCSV(strings.NewReader(csvStr))

	for i := 0; i < df.Nrow(); i++ {
		fmt.Println(df.Elem(i,0))
	}

	a := "sdfdsf\n"
	fmt.Println(a)
}


