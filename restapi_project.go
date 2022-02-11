package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// handleEventSGAPIAddProjectFunc 함수는 샷건에서 프로젝트가 추가되었을 때 샷건 이벤트 핸들러를 통해 프로젝트를 추가하는 함수이다.
func handleEventSGAPIAddProjectFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Post method only", http.StatusMethodNotAllowed)
		return
	}

	// URL에서 id, name, startdate, enddate를 가져온다.
	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
		return
	}
	name := q.Get("name")
	if name == "" {
		http.Error(w, "URL에 name을 입력해주세요", http.StatusBadRequest)
		return
	}
	startDate := q.Get("startdate")
	if startDate == "" {
		http.Error(w, "URL에 startdate를 입력해주세요", http.StatusBadRequest)
		return
	}
	endDate := q.Get("enddate")
	if endDate == "" {
		http.Error(w, "URL에 enddate를 입력해주세요", http.StatusBadRequest)
		return
	}

	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	p := Project{
		ID:        strings.TrimSpace(id),
		Name:      strings.TrimSpace(name),
		StartDate: startDate,
		SMEndDate: endDate,
	}

	var payment Payment
	payment.Expenses, err = encryptAES256Func("0") // 샷건에는 총 매출을 입력할 수 있는 공간이 없기 때문에 임시로 총 매출을 0으로 설정한다.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	p.Payment = append(p.Payment, payment)

	err = p.CheckErrorFunc()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = addProjectFunc(client, p) // 프로젝트 추가
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Token 가져오기
	token, _ := getTokenFromHeaderFunc(w, r)
	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()
	log.Content = fmt.Sprintf("프로젝트 %s가 추가되었습니다.(ShotgunEvent)", id)

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// json으로 결과 전송
	data, err := json.Marshal(p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func handleAPIRmProjectFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Delete method only", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
		return
	}

	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Access Level 확인
	accesslevel, err := getAccessLevelFromHeaderFunc(r, client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if accesslevel < AdminLevel {
		http.Error(w, "삭제 권한이 없는 계정입니다", http.StatusUnauthorized)
		return
	}

	err = rmProjectFunc(client, id) // 프로젝트 삭제
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 해당하는 벤더 삭제
	err = rmVendorFunc(client, id, "", "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Token 가져오기
	token, _ := getTokenFromHeaderFunc(w, r)
	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()
	log.Content = fmt.Sprintf("프로젝트 %s가 삭제되었습니다.", id)

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// json으로 결과 전송
	data, err := json.Marshal(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleMonthlyPurchaseCostFunc 함수는 rest API를 이용하여 date에 해당하는 구매 내역을 반환하는 함수이다.
func handleMonthlyPurchaseCostFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Get method only", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
		return
	}
	date := q.Get("date")
	if date == "" {
		http.Error(w, "URL에 date를 입력해주세요", http.StatusBadRequest)
		return
	}

	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Access Level 확인
	accesslevel, err := getAccessLevelFromHeaderFunc(r, client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if accesslevel < ManagerLevel {
		http.Error(w, "프로젝트 구매비의 읽기 권한이 없는 계정입니다", http.StatusUnauthorized)
		return
	}

	project, err := getProjectFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	purchaseCost := project.SMMonthlyPurchaseCost[date]
	for i, cost := range purchaseCost {
		decryptedCost, err := decryptAES256Func(cost.Expenses)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		purchaseCost[i].Expenses = decryptedCost
	}

	// json으로 결과 전송
	data, err := json.Marshal(purchaseCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleAPISetMonthlyPurchaseCostFunc 함수는 rest API를 이용하여 그 달의 구매 내역을 업데이트하는 함수이다.
func handleAPISetMonthlyPurchaseCostFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Post method only", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
		return
	}
	date := q.Get("date")
	if date == "" {
		http.Error(w, "URL에 date를 입력해주세요", http.StatusBadRequest)
		return
	}
	num := q.Get("num")
	if num == "" {
		http.Error(w, "URL에 num을 입력해주세요", http.StatusBadRequest)
		return
	}

	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Access Level 확인
	accesslevel, err := getAccessLevelFromHeaderFunc(r, client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if accesslevel < ManagerLevel {
		http.Error(w, "프로젝트 구매비의 수정 권한이 없는 계정입니다", http.StatusUnauthorized)
		return
	}

	purchaseCostNum, err := strconv.Atoi(num)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var purchaseCostList []PurchaseCost
	totalExpenses := 0
	for i := 0; i < purchaseCostNum; i++ {
		companyName := q.Get(fmt.Sprintf("companyName%d", i))
		detail := q.Get(fmt.Sprintf("detail%d", i))
		expenses := q.Get(fmt.Sprintf("expenses%d", i))

		if companyName == "" && detail == "" && expenses == "" { // 업체명, 내역, 금액 모두 빈칸이면 continue
			continue
		}
		if companyName == "" && detail == "" {
			http.Error(w, "companyName, detail 둘 중 하나는 필수로 입력해야 합니다", http.StatusBadRequest)
			return
		}
		if expenses == "" {
			http.Error(w, "expenses를 입력해주세요", http.StatusBadRequest)
			return
		}
		excryptedExpenses, err := encryptAES256Func(expenses)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		purchaseCost := PurchaseCost{
			CompanyName: companyName,
			Detail:      detail,
			Expenses:    excryptedExpenses,
		}
		purchaseCostList = append(purchaseCostList, purchaseCost)
		expensesInt, err := strconv.Atoi(expenses)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		totalExpenses = totalExpenses + expensesInt
	}

	project, err := getProjectFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if project.SMMonthlyPurchaseCost == nil { // 비어있다면 초기화를 해준다.
		project.SMMonthlyPurchaseCost = map[string][]PurchaseCost{}
	}
	project.SMMonthlyPurchaseCost[date] = purchaseCostList

	err = setProjectFunc(client, project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// json으로 결과 전송
	data, err := json.Marshal(totalExpenses)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleAPIMonthlyPaymentFunc 함수는 rest API를 이용하여 프로젝트의 월별 매출 내역을 가져오는 함수이다.
func handleAPIMonthlyPaymentFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Get method only", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
		return
	}
	date := q.Get("date")
	if date == "" {
		http.Error(w, "URL에 date를 입력해주세요", http.StatusBadRequest)
		return
	}

	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Access Level 확인
	accesslevel, err := getAccessLevelFromHeaderFunc(r, client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if accesslevel < ManagerLevel {
		http.Error(w, "프로젝트 월별 매출의 읽기 권한이 없는 계정입니다", http.StatusUnauthorized)
		return
	}

	project, err := getProjectFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	payments := project.SMMonthlyPayment[date]
	for i, payment := range payments {
		decryptedPayment, err := decryptAES256Func(payment.Expenses)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		payments[i].Expenses = decryptedPayment
	}

	// json으로 결과 전송
	data, err := json.Marshal(payments)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleAPISetMonthlyPaymentFunc 함수는 rest API를 이용하여 그 달의 매출 내역을 업데이트하는 함수이다.
func handleAPISetMonthlyPaymentFunc(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Post method only", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	id := q.Get("id")
	if id == "" {
		http.Error(w, "URL에 id를 입력해주세요", http.StatusBadRequest)
		return
	}
	date := q.Get("date")
	if date == "" {
		http.Error(w, "URL에 date를 입력해주세요", http.StatusBadRequest)
		return
	}
	num := q.Get("num")
	if num == "" {
		http.Error(w, "URL에 num을 입력해주세요", http.StatusBadRequest)
		return
	}

	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Access Level 확인
	accesslevel, err := getAccessLevelFromHeaderFunc(r, client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if accesslevel < ManagerLevel {
		http.Error(w, "프로젝트 매출의 수정 권한이 없는 계정입니다", http.StatusUnauthorized)
		return
	}

	paymentNum, err := strconv.Atoi(num)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var paymentList []Payment
	totalExpenses := 0
	for i := 0; i < paymentNum; i++ {
		typ := q.Get(fmt.Sprintf("type%d", i))
		date := q.Get(fmt.Sprintf("date%d", i))
		expenses := q.Get(fmt.Sprintf("expenses%d", i))
		status := q.Get(fmt.Sprintf("status%d", i))
		depositDate := q.Get(fmt.Sprintf("depositdate%d", i))

		if typ == "" && date == "" && expenses == "" && status == "" { // 업체명, 내역, 금액 모두 빈칸이면 continue
			continue
		}
		if typ == "" {
			http.Error(w, "type을 입력해주세요", http.StatusBadRequest)
			return
		}
		if date == "" {
			http.Error(w, "date를 입력해주세요", http.StatusBadRequest)
			return
		}
		if expenses == "" {
			http.Error(w, "expenses를 입력해주세요", http.StatusBadRequest)
			return
		}
		if status == "" {
			http.Error(w, "status를 입력해주세요", http.StatusBadRequest)
			return
		}
		excryptedExpenses, err := encryptAES256Func(expenses)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		statusBool, err := strconv.ParseBool(status)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		payment := Payment{
			Type:        typ,
			Date:        date,
			Expenses:    excryptedExpenses,
			Status:      statusBool,
			DepositDate: depositDate,
		}
		paymentList = append(paymentList, payment)
		expensesInt, err := strconv.Atoi(expenses)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		totalExpenses = totalExpenses + expensesInt
	}

	project, err := getProjectFunc(client, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if project.SMMonthlyPayment == nil { // 비어있다면 초기화를 해준다.
		project.SMMonthlyPayment = map[string][]Payment{}
	}
	project.SMMonthlyPayment[date] = paymentList

	err = setProjectFunc(client, project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// json으로 결과 전송
	data, err := json.Marshal(totalExpenses)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// handleAPIUpdateProjectsFunc 함수는 rest API를 이용하여 샷건에 존재하는 프로젝트로 업데이트합니다.
func handleAPIUpdateProjectsFunc(w http.ResponseWriter, r *http.Request) {
	// Method 확인
	if r.Method != http.MethodPost {
		http.Error(w, "Post method only", http.StatusMethodNotAllowed)
		return
	}

	// mongoDB client 연결
	credential := options.Credential{
		Username: *flagDBID,
		Password: *flagDBPW,
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(*flagMongoDBURI).SetAuth(credential))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.Disconnect(ctx)
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// AccessLevel 확인
	accesslevel, err := getAccessLevelFromHeaderFunc(r, client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if accesslevel < ManagerLevel {
		http.Error(w, "업데이트 권한이 없는 계정입니다", http.StatusUnauthorized)
		return
	}

	// AdminSetting을 가져온다.
	adminSetting, err := getAdminSettingFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 샷건에서 프로젝트를 가져온다
	projects, err := sgGetAllProjectsFunc(adminSetting.SGExcludeProjects)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 현재 DB에 저장된 프로젝트를 가져온다.
	nowProjects, err := getAllProjectsFunc(client)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	nowProjectsID := make([]string, len(nowProjects))
	for _, np := range nowProjects {
		nowProjectsID = append(nowProjectsID, np.ID)
	}

	// 샷건에서 가져온 프로젝트 중에 현재 DB에 저장되지 않은 프로젝트를 추가한다.
	for _, p := range projects {
		if checkStringInListFunc(p.ID, nowProjectsID) {
			continue
		}
		err = addProjectFunc(client, p)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Token 가져오기
	token, _ := getTokenFromHeaderFunc(w, r)
	log := Log{}
	log.UserID = token.ID
	log.CreatedAt = time.Now()
	log.Content = "샷건에 존재하는 프로젝트들이 업데이트되었습니다."

	err = addLogsFunc(client, log)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
