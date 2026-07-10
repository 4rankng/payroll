package main
import ("fmt";"os";"github.com/xuri/excelize/v2";excelparser "api-server/internal/app/services/excel")
func main(){
	for _, p := range os.Args[1:] {
		f, err := excelize.OpenFile(p); if err != nil { fmt.Println(err); continue }
		data, perr := excelparser.ParseBCCFile(f)
		if perr != nil { fmt.Printf("%-50s ERR: %v\n", p, perr); f.Close(); continue }
		fmt.Printf("%-50s OK: %d employees, %d shift rates\n", p, len(data.Employees), len(data.ShiftRates))
		// show first employee's entries to confirm day numbers + hours parsed
		if len(data.Employees) > 0 {
			e := data.Employees[0]
			fmt.Printf("    1st emp: %q (%s) — %d entries; sample: ", e.FullName, e.EmployeeCode, len(e.Entries))
			for i := 0; i < 3 && i < len(e.Entries); i++ { fmt.Printf("d%d=%s:%.1fh ", e.Entries[i].DayNum, e.Entries[i].ShiftLabel, e.Entries[i].Hours) }
			fmt.Println()
		}
		// sanity: distinct day numbers present
		days := map[int]bool{}; for _, e := range data.Employees { for _, en := range e.Entries { days[en.DayNum]=true } }
		ds := []int{}; for d := range days { ds = append(ds,d) }; 
		fmt.Printf("    distinct day numbers across all entries: %v\n", ds)
		f.Close()
	}
}
