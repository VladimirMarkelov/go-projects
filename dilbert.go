package main

import (
	// "bufio"
	"fmt"
	ui "github.com/VladimirMarkelov/clui"
	"html"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

func appendToFile(fileName, str string) bool {
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err = f.WriteString(str + "\n"); err != nil {
		panic(err)
	}

	return true
}

func download(c *ui.Composer, y, m, d int, lb *ui.ListBox) bool {
	res, err := http.Get(fmt.Sprintf("http://dilbert.com/fast/%d-%02d-%02d", y, m, d))
	if err != nil {
		add_lb_item(c, lb, "Failed: second attempt")
		time.Sleep(1500 * time.Millisecond)
		res, err = http.Get(fmt.Sprintf("http://dilbert.com/fast/%d-%02d-%02d", y, m, d))
		if err != nil {
			add_lb_item(c, lb, "Failed: third attempt")
			time.Sleep(3000 * time.Millisecond)
			res, err = http.Get(fmt.Sprintf("http://dilbert.com/fast/%d-%02d-%02d", y, m, d))
			if err != nil {
				add_lb_item(c, lb, fmt.Sprintf("Fatal: %v", err))
				return false
			}
		}
	}

	cont, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	if err != nil {
		add_lb_item(c, lb, fmt.Sprintf("Read: %v", err))
		return false
	}

	// r1, _ := regexp.Compile("<meta[^>]*twitter:description[^>]*>")
	// r2, _ := regexp.Compile("<meta[^>]*twitter:image[^>]*>")
	r1, _ := regexp.Compile("<div class=\"comic-item-container[^>]+data-description=\"([^\"]+)\"")
	r2, _ := regexp.Compile("<div class=\"comic-item-container[^>]+data-image=\"([^\"]+)\"")
	// idx1 := r1.Find(cont)
	// idx2 := r2.Find(cont)

	// cnt := string(idx1)
	// img := string(idx2)

	// r1, _ = regexp.Compile("content=\"([^\"]+)\"")
	// sub1 := r1.FindStringSubmatch(cnt)
	// sub2 := r1.FindStringSubmatch(img)
	text := string(cont)
	sub1 := r1.FindStringSubmatch(text)
	sub2 := r2.FindStringSubmatch(text)

	if sub1 != nil {
		s := fmt.Sprintf("%d-%02d-%02d: %s", y, m, d, html.UnescapeString(string(sub1[1])))
		appendToFile("tags.txt", s)
	}
	if sub2 != nil {
		imageLink := html.UnescapeString(string(sub2[1]))
		s := fmt.Sprintf("%d%02d%02d.gif", y, m, d)
		res, err = http.Get(imageLink)
		if err != nil {
			add_lb_item(c, lb, fmt.Sprintf("Image Fatal: %v", err))
			return false
		}
		cont, err = ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			add_lb_item(c, lb, fmt.Sprintf("Image read fail: %v", err))
			return false
		}

		fName := fmt.Sprintf(".\\%d\\%s", y, s)
		add_lb_item(c, lb, fmt.Sprintf("Downloading picture to %s", fName))
		ioutil.WriteFile(fName, cont, 0777)
	}

	ioutil.WriteFile("saved.html", cont, 0777)
	return true
}

func createDir(year int) bool {
	s := fmt.Sprintf("%d", year)
	_, err := os.Stat(s)
	if os.IsNotExist(err) {
		os.Mkdir(s, 0777)
		return true
	}

	return false
}

func itemExists(y, m, d int) bool {
	fname := fmt.Sprintf("%d\\%d%02d%02d.gif", y, y, m, d)
	_, err := os.Stat(fname)
	return err == nil
}

func add_lb_item(c *ui.Composer, lb *ui.ListBox, item string) {
	lb.AddItem(item)
	lb.SelectItem(lb.ItemCount() - 1)
	lb.EnsureVisible()
	c.PutEvent(ui.Event{Type: ui.EventRedraw})
}

func run_download(c *ui.Composer, pb *ui.ProgressBar, lb *ui.ListBox, fromDate, toDate time.Time) {
	s := fmt.Sprintf("%v - %v", fromDate.String(), toDate.String())
	ioutil.WriteFile("Out.htm", []byte(s), 0777)

	un1 := fromDate.Unix() / (60 * 60 * 24)
	un2 := toDate.Unix() / (60 * 60 * 24)

	diff := int(un2 - un1)

	if diff < 0 {
		return
	}

	pb.SetLimits(0, diff)
	pb.SetValue(0)

	d := fromDate
	started := time.Now()
	downloaded := 0
	for i := 0; i <= diff; i++ {
		s := fmt.Sprintf("Processing %d-%02d-%02d...", d.Year(), int(d.Month()), d.Day())
		add_lb_item(c, lb, s)

		if itemExists(d.Year(), int(d.Month()), d.Day()) {
			add_lb_item(c, lb, "Already downloaded - skipping...")
		} else {
			if createDir(d.Year()) {
				add_lb_item(c, lb, fmt.Sprintf("Directory %d created", d.Year()))
			}

			if !download(c, d.Year(), int(d.Month()), d.Day(), lb) {
				add_lb_item(c, lb, "Failed")
				break
			}

			time.Sleep(750 * time.Millisecond)
			add_lb_item(c, lb, "Completed")
			downloaded++

			if downloaded > 10 {
				add_lb_item(c, lb, fmt.Sprintf("Time elapsed: %s", time.Since(started)))
				downloaded = 0
			}
		}
		_ = pb.Step()
		d = d.AddDate(0, 0, 1)
	}
	add_lb_item(c, lb, fmt.Sprintf("Done in %s!", time.Since(started)))
}

func mainLoop() {
	c := ui.InitLibrary()

	defer c.Close()

	// - from -------------- to --------------
	// | Day Month  Year  | Day Month  Year  |
	// | day monthV yearV | day monthV yearV |
	// ---------------------------------------
	// pbpbpbpbpbpbpbp
	// textscroll
	// Close

	view := c.CreateView(1, 1, 15, 5, "Dilbert Downloader")
	view.SetPack(ui.Vertical)

	topFrame := ui.NewFrame(view, view, 3, 3, ui.BorderNone, ui.DoNotScale)

	frmFrom := ui.NewFrame(view, topFrame, 7, 3, ui.BorderSingle, ui.DoNotScale)
	frmFrom.SetPaddings(1, 1, 1, 0)
	frmFrom.SetTitle("From")
	frmFromDay := ui.NewFrame(view, frmFrom, 2, 2, ui.BorderNone, ui.DoNotScale)
	frmFromDay.SetPack(ui.Vertical)
	ui.NewLabel(view, frmFromDay, 3, 1, "Day", ui.DoNotScale)
	fromDay := ui.NewEditField(view, frmFromDay, 3, "16", ui.DoNotScale)
	frmFromMonth := ui.NewFrame(view, frmFrom, 2, 2, ui.BorderNone, ui.DoNotScale)
	frmFromMonth.SetPack(ui.Vertical)
	ui.NewLabel(view, frmFromMonth, 5, 1, "Month", ui.DoNotScale)
	fromMnth := ui.NewEditField(view, frmFromMonth, 5, "04", ui.DoNotScale)
	frmFromYear := ui.NewFrame(view, frmFrom, 2, 2, ui.BorderNone, ui.DoNotScale)
	frmFromYear.SetPack(ui.Vertical)
	ui.NewLabel(view, frmFromYear, 5, 1, "Year", ui.DoNotScale)
	fromYear := ui.NewEditField(view, frmFromYear, 5, "1989", ui.DoNotScale)

	frmTo := ui.NewFrame(view, topFrame, 7, 3, ui.BorderSingle, ui.DoNotScale)
	frmTo.SetPaddings(1, 1, 1, 0)
	frmTo.SetTitle("To")
	frmToDay := ui.NewFrame(view, frmTo, 2, 2, ui.BorderNone, ui.DoNotScale)
	frmToDay.SetPack(ui.Vertical)
	ui.NewLabel(view, frmToDay, 3, 1, "Day", ui.DoNotScale)
	toDay := ui.NewEditField(view, frmToDay, 3, "16", ui.DoNotScale)
	frmToMonth := ui.NewFrame(view, frmTo, 2, 2, ui.BorderNone, ui.DoNotScale)
	frmToMonth.SetPack(ui.Vertical)
	ui.NewLabel(view, frmToMonth, 5, 1, "Month", ui.DoNotScale)
	toMnth := ui.NewEditField(view, frmToMonth, 5, "04", ui.DoNotScale)
	frmToYear := ui.NewFrame(view, frmTo, 2, 2, ui.BorderNone, ui.DoNotScale)
	frmToYear.SetPack(ui.Vertical)
	ui.NewLabel(view, frmToYear, 5, 1, "Year", ui.DoNotScale)
	toYear := ui.NewEditField(view, frmToYear, 5, "1989", ui.DoNotScale)

	pb := ui.NewProgressBar(view, view, 10, 1, ui.DoNotScale)
	pb.SetLimits(0, 10)

	lbox := ui.NewListBox(view, view, 40, 7, 1)

	ui.NewFrame(view, view, 1, 1, ui.BorderNone, ui.DoNotScale)
	frmBtn := ui.NewFrame(view, view, 2, 2, ui.BorderNone, ui.DoNotScale)
	ui.NewFrame(view, frmBtn, 1, 2, ui.BorderNone, 1)
	btnGo := ui.NewButton(view, frmBtn, 9, 4, "Go", ui.DoNotScale)
	ui.NewFrame(view, frmBtn, 1, 2, ui.BorderNone, 1)
	btnQuit := ui.NewButton(view, frmBtn, 9, 4, "Quit", ui.DoNotScale)
	ui.NewFrame(view, frmBtn, 1, 2, ui.BorderNone, 1)

	btnGo.OnClick(func(ev ui.Event) {
		year, _ := strconv.Atoi(fromYear.Title())
		month, _ := strconv.Atoi(fromMnth.Title())
		day, _ := strconv.Atoi(fromDay.Title())
		dateFrom := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		year, _ = strconv.Atoi(toYear.Title())
		month, _ = strconv.Atoi(toMnth.Title())
		day, _ = strconv.Atoi(toDay.Title())
		dateTo := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		var _ = dateFrom
		_ = dateTo
		go run_download(c, pb, lbox, dateFrom, dateTo)
	})

	btnQuit.OnClick(func(ev ui.Event) {
		go c.Stop()
	})

	// c.RefreshScreen(true)
	c.MainLoop()
}

func main() {
	mainLoop()
}
