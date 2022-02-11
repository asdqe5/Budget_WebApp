// 프로젝트 결산 프로그램
//
// Description : http 관련 스크립트

package main

import (
	"html/template"
	"log"
	"net/http"

	"github.com/shurcooL/httpfs/html/vfstemplate"
)

// LoadTemplates 함수는 탬플릿을 로딩하는 함수이다.
func loadTemplatesFunc() (*template.Template, error) {
	t := template.New("").Funcs(funcMap)
	t, err := vfstemplate.ParseGlob(assets, t, "/template/*.html")
	return t, err
}

var funcMap = template.FuncMap{
	// templatefunc.go
	"intToAccessLevelFunc":      intToAccessLevelFunc,
	"addIntFunc":                addIntFunc,
	"stringToDateFunc":          stringToDateFunc,
	"getColorOfRevenueFunc":     getColorOfRevenueFunc,
	"checkLineChangeFunc":       checkLineChangeFunc,
	"splitLineFunc":             splitLineFunc,
	"durationToTimeFunc":        durationToTimeFunc,
	"supDurationToTimeFunc":     supDurationToTimeFunc,
	"hasStatusFunc":             hasStatusFunc,
	"getStatusFunc":             getStatusFunc,
	"getMonthlyPaymentInfoFunc": getMonthlyPaymentInfoFunc,
	"putCommaFunc":              putCommaFunc,
	"changeDateFormatFunc":      changeDateFormatFunc,

	// templatefunc_artist.go
	"workingDayByYearFunc": workingDayByYearFunc,
	"hourlyWageByYearFunc": hourlyWageByYearFunc,

	// templatefunc_cost.go
	"decryptCostFunc":              decryptCostFunc,
	"decryptPaymentFunc":           decryptPaymentFunc,
	"totalOfPurchaseCostFunc":      totalOfPurchaseCostFunc,
	"totalLaborOfCostSumFunc":      totalLaborOfCostSumFunc,
	"totalOfFinishedLaborCostFunc": totalOfFinishedLaborCostFunc,
	"calRatioFunc":                 calRatioFunc,

	// templatefunc_vendor.go
	"lenOfVendorsMapFunc":          lenOfVendorsMapFunc,
	"lenOfVendorsListFunc":         lenOfVendorsListFunc,
	"setVendorInfoMapFunc":         setVendorInfoMapFunc,
	"getVendorTooltipFunc":         getVendorTooltipFunc,
	"calUnitPriceByCutsFunc":       calUnitPriceByCutsFunc,
	"checkMediumPlatingStatusFunc": checkMediumPlatingStatusFunc,

	// templatefunc_bg.go
	"calNegoRatioFunc":     calNegoRatioFunc,
	"getBGPartInfoMapFunc": getBGPartInfoMapFunc,

	// etc
	"listToStringFunc":                listToStringFunc,
	"mapToStringFunc":                 mapToStringFunc,
	"checkStringInListFunc":           checkStringInListFunc,
	"getLastStatusOfProjectFunc":      getLastStatusOfProjectFunc,
	"getThisMonthStatusOfProjectFunc": getThisMonthStatusOfProjectFunc,
	"PreviousPageFunc":                PreviousPageFunc,
	"NextPageFunc":                    NextPageFunc,
	"SplitPageFunc":                   SplitPageFunc,
}

func webServerFunc() {
	vfsTemplate, err := loadTemplatesFunc()
	if err != nil {
		log.Fatal(err)
	}
	TEMPLATES = vfsTemplate

	// 리소스 로딩
	http.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(assets)))

	// 웹주소 설정
	// 오류 제거를 위한 무시
	http.HandleFunc("/favicon.ico", handleIconFunc)

	// 로그인
	http.HandleFunc("/signup", handleSignupFunc)
	http.HandleFunc("/signup-submit", handleSignupSubmitFunc)
	http.HandleFunc("/signup-success", handleSignupSuccessFunc)
	http.HandleFunc("/signin", handleSigninFunc)
	http.HandleFunc("/signin-submit", handleSigninSubmitFunc)
	http.HandleFunc("/signout", handleSignOutFunc)
	http.HandleFunc("/invalidaccess", handleInvalidAccessFunc)

	// profile
	http.HandleFunc("/editprofile", handleEditProfileFunc)
	http.HandleFunc("/editprofile-submit", handleEditProfileSubmitFunc)
	http.HandleFunc("/editprofile-success", handleEditProfileSuccessFunc)
	http.HandleFunc("/updatepassword", handleUpdatePasswordFunc)
	http.HandleFunc("/updatepassword-submit", handleUpdatePasswordSubmitFunc)
	http.HandleFunc("/updatepassword-success", handleUpdatePasswordSuccessFunc)

	// 메인 페이지
	http.HandleFunc("/", handleInitFunc)
	http.HandleFunc("/search", handleSearchFunc)
	http.HandleFunc("/exportinit", handleExportInitFunc)

	// 디테일 페이지
	http.HandleFunc("/detail-sm", handleDetailSMFunc)
	http.HandleFunc("/exportdetailsm", handleExportDetailSMFunc)

	// VFX 타임로그
	http.HandleFunc("/timelog-vfx", handleTimelogVFXFunc)
	http.HandleFunc("/searchtimelog-vfx", handleSearchTimelogVFXFunc)
	http.HandleFunc("/updatetimelog-vfx", handleUpdateTimelogVFXFunc)
	http.HandleFunc("/timelogvfxexcel-download", handleTimelogVFXExcelDownloadFunc)
	http.HandleFunc("/upload-timelogvfxexcel", handleUploadTimelogVFXExcelFunc)
	http.HandleFunc("/timelogvfxexcel-submit", handleTimelogVFXExcelSubmitFunc)
	http.HandleFunc("/updatetimelogvfx-submit", handleUpdateTimelogVFXSubmitFunc)
	http.HandleFunc("/updatetimelogvfx-success", handleUpdateTimelogVFXSuccessFunc)
	http.HandleFunc("/exporttimelog-vfx", handleExportTimelogVFXFunc)

	// CM 타임로그
	http.HandleFunc("/timelog-cm", handleTimelogCMFunc)
	http.HandleFunc("/searchtimelog-cm", handleSearchTimelogCMFunc)
	http.HandleFunc("/updatetimelog-cm", handleUpdateTimelogCMFunc)
	http.HandleFunc("/timelogcmexcel-download", handleTimelogCMExcelDownloadFunc)
	http.HandleFunc("/upload-timelogcmexcel", handleUploadTimelogCMExcelFunc)
	http.HandleFunc("/timelogcmexcel-submit", handleTimelogCMExcelSubmitFunc)
	http.HandleFunc("/updatetimelogcm-submit", handleUpdateTimelogCMSubmitFunc)
	http.HandleFunc("/updatetimelogcm-success", handleUpdateTimelogCMSuccessFunc)
	http.HandleFunc("/exporttimelog-cm", handleExportTimelogCMFunc)

	// 누계 타임로그
	http.HandleFunc("/timelog-total", handleTimelogTotalFunc)
	http.HandleFunc("/searchtimelog-total", handleSearchTimelogTotalFunc)
	http.HandleFunc("/exporttimelog-total", handleExportTimelogTotalFunc)

	// 타임로그
	http.HandleFunc("/finishedtimelog", handleFinishedTimelogFunc)
	http.HandleFunc("/finishedtimelog-submit", handleFinishedTimelogSubmitFunc)

	// 결산 현황
	http.HandleFunc("/smpayment-status", handleSMPaymentStatusFunc)
	http.HandleFunc("/export-smpaymentstatus", handleExportSMPaymentStatusFunc)
	http.HandleFunc("/smvendor-status", handleSMVendorStatusFunc)
	http.HandleFunc("/export-smvendorstatus", handleExportSMVendorStatusFunc)
	http.HandleFunc("/smtotal-status", handleSMTotalStatusFunc)
	http.HandleFunc("/export-smtotalstatus", handleExportSMTotalStatusFunc)

	// 결산 인건비
	http.HandleFunc("/smdetail-laborcost", handleSMDetailLaborCostFunc)
	http.HandleFunc("/export-smdetaillaborcost", handleExportSMDetailLaborCostFunc)
	http.HandleFunc("/smtotal-laborcost", handleSMTotalLaborCostFunc)
	http.HandleFunc("/export-smtotallaborcost", handleExportSMTotalLaborCostFunc)

	// 예산 디테일
	http.HandleFunc("/bg/detail", handleBGDetailFunc)

	// 유저
	http.HandleFunc("/users", handleUsersFunc)
	http.HandleFunc("/update-users", handleUpdateUsersFunc)
	http.HandleFunc("/updateusers-success", handleUpdateUsersSuccess)
	http.HandleFunc("/changepassword", handleChangePasswordFunc)
	http.HandleFunc("/changepassword-submit", handleChangePasswordSubmitFunc)
	http.HandleFunc("/changepassword-success", handleChangePasswordSuccessFunc)

	// 아티스트
	http.HandleFunc("/upload-artistsexcel", handleUploadArtistsExcelFunc)

	// VFX 아티스트
	http.HandleFunc("/artists-vfx", handleArtistsVFXFunc)
	http.HandleFunc("/edit-artistvfx", handleEditArtistVFXFunc)
	http.HandleFunc("/editartistvfx-submit", handleEditArtistVFXSubmitFunc)
	http.HandleFunc("/editartistvfx-success", handleEditArtistVFXSuccessFunc)
	http.HandleFunc("/updateartists-vfx", handleUpdateArtistsVFXFunc)
	http.HandleFunc("/artistsvfxexcel-download", handleArtistsVFXExcelDownloadFunc)
	http.HandleFunc("/artistsvfxexcel-submit", handleArtistsVFXExcelSubmitFunc)
	http.HandleFunc("/updateartistsvfx-submit", handleUpdateArtistsVFXSubmitFunc)
	http.HandleFunc("/updateartistsvfx-success", handleUpdateArtistsVFXSuccessFunc)
	http.HandleFunc("/exportartists-vfx", handleExportArtistsVFXFunc)

	// CM 아티스트
	http.HandleFunc("/artists-cm", handleArtistsCMFunc)
	http.HandleFunc("/edit-artistcm", handleEditArtistCMFunc)
	http.HandleFunc("/editartistcm-submit", handleEditArtistCMSubmitFunc)
	http.HandleFunc("/editartistcm-success", handleEditArtistCMSuccessFunc)
	http.HandleFunc("/updateartists-cm", handleUpdateArtistsCMFunc)
	http.HandleFunc("/artistscmexcel-download", handleArtistsCMExcelDownloadFunc)
	http.HandleFunc("/artistscmexcel-submit", handleArtistsCMExcelSubmitFunc)
	http.HandleFunc("/updateartistscm-submit", handleUpdateArtistsCMSubmitFunc)
	http.HandleFunc("/updateartistscm-success", handleUpdateArtistsCMSuccessFunc)
	http.HandleFunc("/exportartists-cm", handleExportArtistsCMFunc)

	// SUP 타임로그
	http.HandleFunc("/timelogs-sup", handleTimelogSUPFunc)
	http.HandleFunc("/editsuptimelogs-submit", handleEditSUPTimelogFunc)
	http.HandleFunc("/editsuptimelogs-success", handleEditSUPTimelogSuccessFunc)

	// 프로젝트 - 결산
	http.HandleFunc("/projects", handleProjectsFunc)
	http.HandleFunc("/searchprojects", handleSearchProjectsFunc)
	http.HandleFunc("/addproject", handleAddProjectFunc)
	http.HandleFunc("/addproject-submit", handleAddProjectSubmitFunc)
	http.HandleFunc("/addproject-success", handleAddProjectSuccessFunc)
	http.HandleFunc("/edit-projectsm", handleEditProjectSMFunc)
	http.HandleFunc("/editprojectsm-submit", handleEditProjectSMSubmitFunc)
	http.HandleFunc("/editprojectsm-success", handleEditProjectSMSuccessFunc)
	http.HandleFunc("/exportprojects", handleExportProjectsFunc)

	// 프로젝트 - 예산
	http.HandleFunc("/bgprojects", handleBGProjectsFunc)
	http.HandleFunc("/searchbgprojects", handleSearchBGProjectsFunc)
	http.HandleFunc("/addbgproject", handleAddBGProjectFunc)
	http.HandleFunc("/addbgproject-submit", handleAddBGProjectSubmitFunc)
	http.HandleFunc("/addbgproject-success", handleAddBGProjectSuccessFunc)
	http.HandleFunc("/edit-bgproject", handleEditBGProjectFunc)
	http.HandleFunc("/editbgproject-submit", handleEditBGProjectSubmitFunc)
	http.HandleFunc("/editbgproject-success", handleEditBGProjectSuccessFunc)
	http.HandleFunc("/exportbgprojects", handleExportBGProjectsFunc)
	http.HandleFunc("/bgproject-teamsetting", handleBGProjectTSFunc)
	http.HandleFunc("/bgproject-teamsetting-submit", handleBGProjectTSSubmitFunc)
	http.HandleFunc("/bgproject-teamsetting-success", handleBGProjectTSSuccessFunc)

	// 샷, 어셋 - 예산
	http.HandleFunc("/shotasset", handelShotAssetFunc)
	http.HandleFunc("/searchshotasset", handleSearchShotAssetFunc)
	http.HandleFunc("/uploadshot", handleUploadShotFunc)
	http.HandleFunc("/shotexcel-download", handleShotExcelDownloadFunc)
	http.HandleFunc("/upload-shotexcel", handleUploadShotExcelFunc)
	http.HandleFunc("/shotexcel-submit", handleShotExcelSubmitFunc)
	http.HandleFunc("/uploadshot-submit", handleUploadShotSubmitFunc)
	http.HandleFunc("/uploadshot-success", handleUploadShotSuccessFunc)
	http.HandleFunc("/detail-shot", handleDetailShotFunc)
	http.HandleFunc("/exportdetailshot", handleExportDetailShotFunc)
	http.HandleFunc("/uploadasset", handleUploadAssetFunc)
	http.HandleFunc("/assetexcel-download", handleAssetExcelDownloadFunc)
	http.HandleFunc("/upload-assetexcel", handleUploadAssetExcelFunc)
	http.HandleFunc("/assetexcel-submit", handleAssetExcelSubmitFunc)
	http.HandleFunc("/uploadasset-submit", handleUploadAssetSubmitFunc)
	http.HandleFunc("/uploadasset-success", handleUploadAssetSuccessFunc)
	http.HandleFunc("/detail-asset", handleDetailAssetFunc)
	http.HandleFunc("/exportdetailasset", handleExportDetailAssetFunc)

	// 벤더 관리
	http.HandleFunc("/vendors", handleVendorsFunc)
	http.HandleFunc("/searchvendors", handleSearchVendorsFunc)
	http.HandleFunc("/addvendor-page", handleAddVendorPageFunc)
	http.HandleFunc("/addvendor", handleAddVendorFunc)
	http.HandleFunc("/addvendor-submit", handleAddVendorSubmitFunc)
	http.HandleFunc("/addvendor-success", handleAddVendorSuccessFunc)
	http.HandleFunc("/edit-vendor", handleEditVendorFunc)
	http.HandleFunc("/editvendor-submit", handleEditVendorSubmitFunc)
	http.HandleFunc("/editvendor-success", handleEditVendorSuccessFunc)
	http.HandleFunc("/exportvendors", handleExportVendorsFunc)

	// Team Setting
	http.HandleFunc("/bgteamsetting", handleBGTeamSettingFunc)
	http.HandleFunc("/bgteamsetting-submit", handleBGTeamSettingSubmitFunc)
	http.HandleFunc("/bgteamsetting-success", handleBGTeamSEttingSuccessFunc)

	// admin setting
	http.HandleFunc("/adminsetting", handleAdminSettingFunc)
	http.HandleFunc("/adminsetting-submit", handleAdminSettingSubmitFunc)
	http.HandleFunc("/adminsetting-success", handleAdminSettingSuccessFunc)

	// Help
	http.HandleFunc("/help", handleHelpFunc)

	// 로그 페이지
	http.HandleFunc("/log", handleLogFunc)

	// 유저 restAPI
	http.HandleFunc("/api/rmuser", handleAPIRmUserFunc)

	// 아티스트 restAPI
	http.HandleFunc("/api/addartistvfx", handleAPIAddArtistVFXFunc)
	http.HandleFunc("/api/addartistcm", handleAPIAddArtistCMFunc)
	http.HandleFunc("/api/rmartist", handleAPIRmArtistFunc)
	http.HandleFunc("/api/shotgunevent/humanuser/new", handleEventSGAPIAddArtistVFXFunc)

	// 타임로그 restAPI
	http.HandleFunc("/api/checkmonthlystatus", handleAPICheckMonthlyStatusFunc)
	http.HandleFunc("/api/updatetimelog", handleAPIUpdateTimelogFunc)
	http.HandleFunc("/api/rmtimelogbyid", handleAPIRmTimelogByIDFunc)
	http.HandleFunc("/api/rmtimelogbyproject", handleAPIRmTimelogByProjectFunc)
	http.HandleFunc("/api/resettimelog", handleAPIResetTimelogFunc)

	// 프로젝트 restAPI
	http.HandleFunc("/api/rmproject", handleAPIRmProjectFunc)
	http.HandleFunc("/api/monthlyPurchaseCost", handleMonthlyPurchaseCostFunc)
	http.HandleFunc("/api/setMonthlyPurchaseCost", handleAPISetMonthlyPurchaseCostFunc)
	http.HandleFunc("/api/monthlyPayment", handleAPIMonthlyPaymentFunc)
	http.HandleFunc("/api/setMonthlyPayment", handleAPISetMonthlyPaymentFunc)
	http.HandleFunc("/api/updateprojects", handleAPIUpdateProjectsFunc)
	http.HandleFunc("/api/shotgunevent/project/new", handleEventSGAPIAddProjectFunc)

	// 예산 프로젝트 restAPI
	http.HandleFunc("/api/rmbgproject", handleAPIRmBGProjectFunc)

	// Vendor restAPI
	http.HandleFunc("/api/rmvendor", handleAPIRmVendorFunc)

	// Shotgun restAPI
	http.HandleFunc("/api/sgartist", handleAPISGArtistFunc)

	// Admin setting restAPI
	http.HandleFunc("/api/vfxteams", handleAPIVFXTeamsFunc)
	http.HandleFunc("/api/totalteams", handleAPITotalTeamsFunc)

	// 웹서버 실행
	err = http.ListenAndServe(*flagHTTPPort, nil)
	if err != nil {
		log.Fatal(err)
	}
}
