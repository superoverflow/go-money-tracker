package main

import (
	"reflect"
	"testing"
)

func TestGetInvestmentTable(t *testing.T) {
	tablehtml := `
	<table>
		<tr>
			<td>a1</td>
			<td><div>b1</div></td>
			<td>c1</td>
		</tr>
		<tr>
			<td>a2</td>
			<td>b2</td>
			<td>c2</td>
		</tr>
	</table>
	`
	tableHtml := string(tablehtml)
	expect := [][]string{{"a1", "b1", "c1"}, {"a2", "b2", "c2"}}
	data := ExtractTableValues(tableHtml)
	if !reflect.DeepEqual(expect, data) {
		t.Error("Result not Equal")
	}
}

func TestTrimCellText(t *testing.T) {
	result := TrimCellText(" FTSE 100 UCITS ETF (VUKE) actionsTop-upSellSwitch")
	expected := "FTSE 100 UCITS ETF (VUKE)"
	if expected != result {
		t.Error("Result not Equal")
	}
}
