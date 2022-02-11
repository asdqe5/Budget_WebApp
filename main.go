// 프로젝트 결산 프로그램
//
// Description : 메인 스크립트

package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/user"
	"strings"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	TEMPLATES = template.New("")

	flagAdd    = flag.String("add", "", "add artistvfx/artistcm/timelog/project/vendor/user")
	flagRm     = flag.String("rm", "", "rm artist/timelog/project/vendor")
	flagGet    = flag.String("get", "", "get artist/timelog/project")
	flagSet    = flag.String("set", "", "set monthlystatus/project")
	flagSearch = flag.String("search", "", "search artist/timelog/project/vendor")

	// 아티스트 관련 플래그
	flagSetResination = flag.Bool("set-resination", false, "set resination of artists mode")

	// 타임로그 관련 플래그
	flagSubTimelog    = flag.Bool("sub-timelog", false, "subtract duration of timelog mode")
	flagUpdateTimelog = flag.Bool("update-timelog", false, "update timelog mode")
	flagResetTimelog  = flag.Bool("reset-timelog", false, "reset all timelog mode")

	// 프로젝트 관련 플래그
	flagUpdateProject = flag.Bool("update-project", false, "update project(new struct)")

	flagGenKey = flag.Bool("gen-key", false, "generate AES 256 key file mode")

	flagID             = flag.String("id", "", "shotgun id / user id / project id")
	flagName           = flag.String("name", "", "user name / artitst name / project name / vendor name")
	flagDept           = flag.String("dept", "", "dept")
	flagTeam           = flag.String("team", "", "team")
	flagSalary         = flag.String("salary", "", "salary")
	flagResination     = flag.Bool("resination", false, "resination")
	flagYear           = flag.Int("year", 0, "year")
	flagQuarter        = flag.Int("quarter", 0, "quarter")
	flagMonth          = flag.Int("month", 0, "month")
	flagProject        = flag.String("project", "", "project")
	flagDuration       = flag.Float64("duration", -1.0, "duration")
	flagPayment        = flag.Int("payment", 0, "project payment / vendor expenses")
	flagDate           = flag.String("date", "", "date")
	flagStartDate      = flag.String("startdate", "", "start date / vendor downpayment date")
	flagEndDate        = flag.String("enddate", "", "end date / vendor balance date")
	flagIsFinished     = flag.Bool("isfinished", false, "project is finished")
	flagTotalAmount    = flag.Int("totalamount", 0, "project total amount")
	flagLaborCost      = flag.Int("laborcost", 0, "project labor cost")
	flagProgressCost   = flag.Int("progresscost", 0, "project progress cost")
	flagPurchaseCost   = flag.Int("purchasecost", 0, "project purchase cost")
	flagRenderFarmCost = flag.Int("renderfarmcost", 0, "project renderfarm cost")
	flagPW             = flag.String("password", "", "user password")
	flagAccessLevel    = flag.Int("accesslevel", 0, "user access level")
	flagStatus         = flag.String("status", "false", "status")
	flagType           = flag.String("type", "sm", "sm/bg")
	flagCuts           = flag.Int("cuts", 0, "vendor cuts")
	flagTasks          = flag.String("tasks", "", "vendor tasks")

	// 서비스에 필요한 인수
	flagPagenum    = flag.Int64("pagenum", 20, "maximum number of logs in a page")
	flagMongoDBURI = flag.String("mongodburi", "mongodb://localhost:27017", "mongoDB URI(ex.mongodb://localhost:27017")
	flagDBIP       = flag.String("dbip", "", "DB IP")
	flagDBName     = flag.String("dbname", "budget", "DB name")
	flagHTTPPort   = flag.String("http", "", "Web Service Port Number.")
	flagCookieAge  = flag.Int("cookieage", 4, "cookie age (hour)")             // MPAA 기준 4시간이다.
	flagDBID       = flag.String("dbid", "", "mongoDB Authorization ID")       // mongoDB 권한 아이디
	flagDBPW       = flag.String("dbpw", "", "mongoDB Authorization Password") // mongoDB 권한 비밀번호
)

func main() {
	// 로그 설정
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("budget: ")

	flag.Parse()
	user, err := user.Current()
	if err != nil {
		log.Fatal()
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter DB AuthUsername: ")
	username, _ := reader.ReadString('\n')
	fmt.Print("Enter DB AuthPassword: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Println("\nPassword typed: " + string(bytePassword))
	}
	fmt.Println("\nMongoDB Authoization ID: " + username)
	password := string(bytePassword)
	*flagDBID = strings.TrimSpace(username)
	*flagDBPW = strings.TrimSpace(password)

	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}

	if *flagAdd == "artistvfx" {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}
		addArtistVFXCmdFunc()
	} else if *flagAdd == "artistcm" {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}
		addArtistCMCmdFunc()
	} else if *flagAdd == "timelog" {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}
		addTimelogCmdFunc()
	} else if *flagAdd == "user" {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}
		addUserCmdFunc()
	} else if *flagAdd == "project" {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}
		addProjectCmdFunc()
	} else if *flagAdd == "vendor" {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}
		addVendorCmdFunc()
	} else if *flagRm == "artist" {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}
		rmArtistCmdFunc()
	} else if *flagRm == "timelog" {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}
		rmTimelogCmdFunc()
	} else if *flagRm == "project" {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}
		rmProjectCmdFunc()
	} else if *flagRm == "vendor" {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}
		rmVendorCmdFunc()
	} else if *flagSubTimelog {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}
		subTimelogCmdFunc()
	} else if *flagGet == "artist" {
		getArtistCmdFunc()
	} else if *flagGet == "timelog" {
		getTimelogCmdFunc()
	} else if *flagGet == "project" {
		getProjectCmdFunc()
	} else if *flagSet == "monthlystatus" {
		setMonthlyStatusCmdFunc()
	} else if *flagSet == "project" {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}
		setProjectCmdFunc()
	} else if *flagSearch == "artist" {
		searchArtistCmdFunc()
	} else if *flagSearch == "timelog" {
		searchTimelogCmdFunc()
	} else if *flagSearch == "project" {
		searchProjectCmdFunc()
	} else if *flagSearch == "vendor" {
		searchVendorCmdFunc()
	} else if *flagUpdateTimelog {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}
		updateTimelogCmdFunc()
	} else if *flagResetTimelog {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}
		resetAllTimelogCmdFunc()
	} else if *flagSetResination {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}
		setResinationCmdFunc()
	} else if *flagUpdateProject {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}
		updateProjectCmdFunc()
	} else if *flagGenKey {
		// root 계정인지 확인
		if user.Username != "root" {
			log.Fatal(errors.New("root 권한이 필요합니다"))
		}

		// key 파일이 존재하는지 확인
		existed, err := checkAESKeyFileFunc()
		if err != nil {
			log.Fatal(err)
		}
		if existed {
			log.Fatal(errors.New("이미 key 파일이 존재합니다"))
		}

		err = genAESKEYFileFunc()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Generated successfully\n")
	} else if *flagHTTPPort != "" {
		ip, err := serviceIPFunc()
		if err != nil {
			log.Fatal(err)
		}

		serviceFunc() // 서비스 실행

		fmt.Printf("Service start: http://%s\n", ip)
		webServerFunc()
	} else {
		flag.PrintDefaults()
		os.Exit(1)
	}
}
