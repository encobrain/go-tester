package tester

import (
	"testing"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"os/exec"
	"regexp"
	"io/ioutil"

	"strings"
	"strconv"
	"time"
)

type Tester struct {
	ColorSheme    *ColorSheme
	Tab           string
	Filter        *regexp.Regexp
	ShowIgnored   bool
	TestRuns      int
	SaveAllLogs   bool
	Restarts      int
	FreezeTimeout time.Duration

	RootPath	string
	LogsPath   	string

	t 			*testing.T
	passed 		int
	failed 		int
	ignored 	int
	showedDirs  map[string]bool
}

func (t *Tester) showDir (dir string) {
	dir = strings.Replace(dir, t.RootPath + string(filepath.Separator), "",-1)

	parts := strings.Split(dir, string(filepath.Separator))

	dir = ""

	for i,part := range parts {
		dir += part + string(filepath.Separator)

		if !t.showedDirs[dir] {
			t.showedDirs[dir] = true
			fmt.Printf("%s%s\n", strings.Repeat(t.Tab,i), t.ColorSheme.Folder.Sprint(part))
		}
	}
}

func (t *Tester) runTestsInDir (dir string, tab string) {
	tests,err := getTests(dir)

	if err != nil { t.t.Fatalf("Get tests in dir %s error: %s", dir, err) }

	for _,testName := range tests {
		fullTestName := dir + ":" + testName

		match := t.Filter.MatchString(fullTestName)

		if !match{ t.ignored++ }

		if match || t.ShowIgnored {
			t.showDir(dir)
			fmt.Print(tab, t.ColorSheme.TestName.Sprint(testName), " ")

			if match {
				stdout,stderr,terr := t.runTest(dir, testName)

				if len(stderr) != 0 {
					stdout = append(stdout, []byte("\nstderr:\n")...)
					stdout = append(stdout, stderr...)
				}

				logdir := strings.Replace(dir, t.RootPath, t.LogsPath, 1)

				err = os.MkdirAll(logdir, 0777)

				if err != nil { t.t.Fatalf("Cant create dir %s : %s", logdir, err) }

				logfileName := filepath.Join(logdir, testName+".log")

				werr := ioutil.WriteFile(logfileName, stdout, 0644)

				if werr != nil { t.t.Fatalf("\nWrite log file %s error: %s", logfileName, werr) }

				result := t.ColorSheme.Pass.Sprint("✔ Passed    ")

				if terr != nil {
					t.failed++
					result = t.ColorSheme.Fail.Sprint("✘ Failed    ")
				} else {
					t.passed++
				}

				fmt.Println(result)

			} else {
				fmt.Println(t.ColorSheme.Ignore.Sprint("⊝ Ignored    "))
			}
		}
	}

	files,err := ioutil.ReadDir(dir)

	if err != nil {
		t.t.Fatalf("Read dir %s error: %s", dir, err)
	}

	for _,fi := range files {
		if fi.IsDir() && fi.Name()[0] != '.' { t.runTestsInDir(filepath.Join(dir, fi.Name()), tab+t.Tab) }
	}
}

var defaultFilter = regexp.MustCompile(".")

// Runs tests. Ignored all tests with comments "@Tester:ignore" before test name
func (t *Tester) Test (T *testing.T) {
	if t.ColorSheme == nil { t.ColorSheme = DefaultColorSheme }
	if t.Tab == "" { t.Tab = "   " }
	if t.Filter == nil { t.Filter = defaultFilter }
	if t.TestRuns == 0 { t.TestRuns = 1 }
	if t.LogsPath == "" { t.LogsPath = "./tests-logs" }
	if t.RootPath == "" { t.RootPath = "./" }

	_,callerFile,_,ok := runtime.Caller(1)
	if !ok { T.Fatalf("Cant get filepath") }

	if !filepath.IsAbs(t.RootPath) {
		t.RootPath = filepath.Join(callerFile, "../", t.RootPath)
	}

	if !filepath.IsAbs(t.LogsPath) {
		t.LogsPath = filepath.Join(callerFile, "../", t.LogsPath)
	}

	t.t = T
	t.passed = 0
	t.failed = 0
	t.ignored = 0
	t.showedDirs = map[string]bool{}
	
	t.runTestsInDir(t.RootPath, "")

	fmt.Printf("\n%s%s%s\n",
		t.ColorSheme.Pass.Sprint(fmt.Sprintf("Passed: %d    ", t.passed)),
		t.ColorSheme.Fail.Sprint(fmt.Sprintf("Failed: %d    ", t.failed)),
		t.ColorSheme.Ignore.Sprint(fmt.Sprintf("Ignored: %d    ", t.ignored)),
	)

	if t.failed > 0 { T.Fail() }
}

// should call in init() func
func ParseDefaultFlags () (tester *Tester) {
	filter  	:= flag.String("filter", ".", "Regexp filter")
	help 		:= flag.Bool("help", false, "Show help")
	ignored 	:= flag.Bool("ignored", false, "Show ignored tests")
	runs 		:= flag.Int("runs", 1, "Test runs")
	allpassed   := flag.Bool("allpassed", false, "Save all logs")
	logsPath    := flag.String("logspath", "./tests-logs", "Logs path")
	freezeTime  := flag.String("freezeTimeout", "10s", "Timeout for mark test fail if it freeze")


	flag.Parse()

	if *help {
		fmt.Println(" -help\n\tPrints this help")
		fmt.Println(" -filter RegExp\n\tRegular expression for filter tests. Default \".\"")
		fmt.Println(" -ignored\n\tShow ignored tests")
		fmt.Println(" -runs n\n\tTest runs count. Default 1")
		fmt.Println(" -allpassed\n\tAlways save all logs. Else last passed or all fails")
		fmt.Println(" -logspath path\n\tPath for logs of test results. Default \"./tests-logs\"")
		fmt.Println(" -freezeTimeout time\nTimeout for mark test fail if it freeze. Default 10s")
		os.Exit(0)
	}

	testFilterRe,err := regexp.Compile(*filter)

	if err != nil {
		fmt.Printf("Regexp incorrect: %s\n", err)
		os.Exit(0)
	}

	ft,err := time.ParseDuration(*freezeTime)
	if err != nil {
		fmt.Printf("Freeze timeout invalid: %s", err)
		os.Exit(0)
	}
	
	tester = &Tester{
		Filter: 		testFilterRe,
		ShowIgnored: 	*ignored,
		TestRuns:		*runs,
		SaveAllLogs: 	*allpassed,
		LogsPath: 		*logsPath,
		FreezeTimeout:  ft,
	}

	return 
}

var testFileNameRe = regexp.MustCompile("_test\\.go$")
var testFuncRe 	   = regexp.MustCompile("(@Tester:ignore.*?[\n\r]+)?\\s*func\\s+(Test\\w*)")

func getTests (path string) (tests []string, err error) {
	files,err := ioutil.ReadDir(path)

	if err != nil { return }

	for _,fi := range files {
		if !fi.IsDir() && testFileNameRe.MatchString(fi.Name()) {
			content,err := ioutil.ReadFile(filepath.Join(path, fi.Name()))

			if err != nil { return nil, err }
			
			for _,m := range testFuncRe.FindAllStringSubmatch(string(content),-1) {
				if m[2] != "TestMain" && m[1] == "" { tests = append(tests, m[2]) }
			}
		}
	}

	return
}

func (t *Tester) runTest (dir string, testName string) (stdout []byte, stderr []byte, err error) {
	var cmd *exec.Cmd

	defer func() {
		if err == nil { return }

		if cmd.Process != nil { cmd.Process.Kill() }
	}()

	var stdoutbuf,stderrbuf *buffer
	runs := t.TestRuns

	freezed := time.NewTimer(t.FreezeTimeout)

	stdoutbuf = &buffer{timer: freezed, timerTimeout: t.FreezeTimeout}
	stderrbuf = &buffer{timer: freezed, timerTimeout: t.FreezeTimeout}

	if t.TestRuns > 1 {
		stdoutbuf.passColor = t.ColorSheme.Pass
		stdoutbuf.failColor = t.ColorSheme.Fail
		stderrbuf.passColor = t.ColorSheme.Pass
		stderrbuf.failColor = t.ColorSheme.Fail
	}

	for {
		cmd = exec.Command("go", "test",
			"-v",
			"-count", strconv.Itoa(runs),
			"-run", "^"+testName+"$",
		)

		cmd.Dir = dir

		cmd.Stdout = stdoutbuf
		cmd.Stderr = stderrbuf

		err = cmd.Start()

		if err != nil { return nil, nil, err }

		done := make(chan int)

		go func() {
			err = cmd.Wait()

			close(done)
		}()

		select {
			case <-freezed.C:
				cmd.Process.Kill()
				stdoutbuf.Write([]byte(fmt.Sprintf("--- FAIL: Test freezed %v\n", t.FreezeTimeout)))
				runs = t.TestRuns - stdoutbuf.i

				if runs != 0 { continue }

			case <-done:
		}

		break
	}

	if err == nil && stdoutbuf.fail>0 { err = fmt.Errorf("Test fail") }

	if t.TestRuns>1 {
		if stdoutbuf.i > 0 { fmt.Print(strings.Repeat("\b", len(strconv.Itoa(stdoutbuf.i)) )) }
		if err != nil && stdoutbuf.pass>0 {
			fmt.Printf("%s.%s ", t.ColorSheme.Pass.Sprint(stdoutbuf.pass), t.ColorSheme.Fail.Sprint(stdoutbuf.fail) )
		}
	}

	if err == nil && !t.SaveAllLogs { stdoutbuf.bytes = stdoutbuf.bytes[stdoutbuf.lastruni:] }

	return stdoutbuf.bytes, stderrbuf.bytes, err
}