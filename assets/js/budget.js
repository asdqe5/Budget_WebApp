// 프로젝트 결산 프로그램
//
// Description : html 이벤트 핸들러

/* 유저 관련 함수 */
// setRmUserModalFunc 함수는 유저 삭제 버튼을 클릭하면 ID를 받아 modal 창에 보여주는 함수이다.
function setRmUserModalFunc(id) {
    document.getElementById("modal-rmuser-id").value = id;
}

// rmUserFunc 함수는 del 버튼을 클릭하면 확인창을 띄우고 ok를 클릭하면 restAPI를 이용하여 유저를 삭제하는 함수이다.
function rmUserFunc(id) {
    let token = document.getElementById("token").value;

    $.ajax({
        url: `/api/rmuser?id=${id}`,
        type: "delete",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        success: function() {
            alert("사용자가 삭제되었습니다.")
            window.location.reload();  // 페이지 새로고침
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}`);
        }
    })
}

// signupPageBlankCheckFunc 함수는 signup 페이지에서 submit 하기 전에 빈 칸이 있는지 체크하는 함수이다.
function signupPageBlankCheckFunc() {
    if (document.getElementById("ID").value == "") {
        alert("ID를 입력해주세요");
        return false;
    }
    if (document.getElementById("Password").value == "") {
        alert("비밀번호를 입력해주세요");
        return false;
    }
    if (document.getElementById("ConfirmPassword").value == "") {
        alert("비밀번호를 입력해주세요");
        return false;
    }
    if (document.getElementById("Password").value != document.getElementById("ConfirmPassword").value) {
        alert("입력받은 2개의 패스워드가 서로 다릅니다");
        return false;
    }
    if (document.getElementById("Name").value == "") {
        alert("이름을 입력해주세요");
        return false;
    }
    if (document.getElementById("Team").value == "") {
        alert("팀을 입력해주세요");
        return false;
    }
}

// profilePageBlankCheckFunc 함수는 edit profile 페이지에서 submit 하기 전에 빈 칸이 있는지 체크하는 함수이다.
function profilePageBlankCheckFunc() {
    if (document.getElementById("team").value == "") {
        alert("팀을 입력해주세요");
        return false;
    }
    if (document.getElementById("name").value == "") {
        alert("이름을 입력해주세요");
        return false;
    }
}

// updatePasswordPageBlankCheckFunc 함수는 update password 페이지에서 submit 하기 전에 빈 칸이 있는지 체크하는 함수이다.
function updatePasswordPageBlankCheckFunc() {
    if (document.getElementById("nowPassword") != null) {
        if (document.getElementById("nowPassword").value == "") {
            alert("현재 사용중인 패스워드를 입력해주세요");
            return false;
        }
    }
    if (document.getElementById("newPassword").value == "") {
        alert("새 패스워드를 입력해주세요");
        return false;
    }
    if (document.getElementById("confirmNewPassword").value == "") {
        alert("확인 패스워드를 입력해주세요");
        return false;
    }
    if (document.getElementById("newPassword").value != document.getElementById("confirmNewPassword").value) {
        alert("입력받은 2개의 패스워드가 서로 다릅니다");
        return false;
    }
}

/* 아티스트 관련 함수 */
// addArtistVFXFunc 함수는 modal-addartistvfx에서 ADD 버튼을 클릭하면 restAPI를 이용하여 아티스트를 추가하는 함수이다.
function addArtistVFXFunc(id, salary, startday, endday) {
    let token = document.getElementById("token").value;
    if (id == "") {
        alert("아티스트 Shotgun ID를 입력해주세요")
        return false
    }
    if (salary == "") {
        alert("아티스트 연봉 정보를 입력해주세요")
        return false
    }
    if (startday == "") {
        alert("아티스트 입사일을 입력해주세요")
        return false
    }
    var change = document.getElementById("modal-addartistvfx-change")
    var changedate = document.getElementById("modal-addartistvfx-changedate").value
    var changesalary = document.getElementById("modal-addartistvfx-changesalary").value
    if (change.checked == true) {
        if (changedate == "") {
            alert("연봉이 바뀐 날짜를 입력해주세요")
            return false
        }
        if (changesalary == "") {
            alert("바뀌기 전 연봉을 입력해주세요")
            return false
        }
    }

    $.ajax({
        url: `/api/addartistvfx?id=${id}&salary=${salary}&startday=${startday}&endday=${endday}&change=${change.checked}&changedate=${changedate}&changesalary=${changesalary}`,
        type: "post",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        success: function(data) {
            alert("아티스트가 추가되었습니다.")
            window.location.reload();  // 페이지 새로고침
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}`);
        }
    })
}

// addArtistCMFunc 함수는 modal-addartistcm에서 ADD 버튼을 클릭하면 restAPI를 이용하여 아티스트를 추가하는 함수이다.
function addArtistCMFunc(id, team, name, salary, startday, endday) {
    let token = document.getElementById("token").value;
    if (id == "") {
        alert("아티스트 ID를 입력해주세요")
        return false
    }
    if (team == "") {
        alert("아티스트 Team을 입력해주세요")
        return false    
    }
    if (name == "") {
        alert("아티스트 Name을 입력해주세요")
        return false    
    }
    if (salary == "") {
        alert("아티스트 연봉 정보를 입력해주세요")
        return false
    }
    if (startday == "") {
        alert("아티스트 입사일을 입력해주세요")
        return false
    }
    var change = document.getElementById("modal-addartistcm-change")
    var changedate = document.getElementById("modal-addartistcm-changedate").value
    var changesalary = document.getElementById("modal-addartistcm-changesalary").value
    if (change.checked == true) {
        if (changedate == "") {
            alert("연봉이 바뀐 날짜를 입력해주세요")
            return false
        }
        if (changesalary == "") {
            alert("바뀌기 전 연봉을 입력해주세요")
            return false
        }
    }
    $.ajax({
        url: `/api/addartistcm?id=${id}&team=${team}&name=${name}&salary=${salary}&startday=${startday}&endday=${endday}&change=${change.checked}&changedate=${changedate}&changesalary=${changesalary}`,
        type: "post",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        success: function(data) {
            alert("아티스트가 추가되었습니다.")
            window.location.reload();  // 페이지 새로고침
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}`);
        }
    })
}

// editArtistPageBlankCheckFunc 함수는 아티스트 edit 페이지에서 submit하기 전에 빈 칸이 있는지 체크하는 함수이다.
function editArtistPageBlankCheckFunc() {
    if (document.getElementById("id").value == "") {
        alert("ID를 입력해주세요");
        return false;
    }
    if (document.getElementById("dept").value == "") {
        alert("Dept를 입력해주세요");
        return false;
    }
    if (document.getElementById("team").value == "") {
        alert("Team을 입력해주세요");
        return false;
    }
    if (document.getElementById("name").value == "") {
        alert("Name을 입력해주세요");
        return false;
    }
    if (document.getElementById("startday").value == "") {
        alert("입사일을 입력해주세요");
        return false
    }
    var startday = new Date(document.getElementById("startday").value)
    if (document.getElementById("endday").value != "") {
        var endday = new Date(document.getElementById("endday").value)
        if (startday > endday) {
            alert("퇴사일이 잘못 입력되었습니다.")
            return false
        }
    }
    if (document.getElementById("changedate").value != "" && document.getElementById("changesalary").value == "") {
        alert("동일 연도 연봉 변경 전 연봉을 입력해주세요.")
        return false
    }
    if (document.getElementById("changedate").value == "" && document.getElementById("changesalary").value != "") {
        alert("동일 연도 연봉 변경 날짜를 입력해주세요.")
        return false
    }
}

// setRmArtistModalFunc 함수는 아티스트 삭제 버튼을 클릭하면 ID, 팀, 이름을 받아 modal 창에 보여주는 함수이다.
function setRmArtistModalFunc(id, team, name) {
    document.getElementById("modal-rmartist-id").value = id;
    document.getElementById("modal-rmartist-team").value = team;
    document.getElementById("modal-rmartist-name").value = name;
}

// rmArtistFunc 함수는 restAPI를 이용하여 아티스트를 삭제하는 함수이다.
function rmArtistFunc(id) {
    let token = document.getElementById("token").value;

    $.ajax({
        url: `/api/rmartist?id=${id}`,
        type: "delete",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        success: function(data) {
            alert("아티스트가 삭제되었습니다.")
            location.reload();  // 페이지 새로고침
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}`);
        }
    })
}

// getSGArtistFunc 함수는 VFX팀 아티스트 edit 페이지에서 Shotgun 정보를 불러와 페이지를 업데이트하는 함수이다.
function getSGArtistFunc(id) {
    let token = document.getElementById("token").value;

    $.ajax({
        url: `/api/sgartist?id=${id}`,
        type: "get",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        async: false,
        success: function(data) {
            document.getElementById("dept").value = data["Dept"];
            document.getElementById("team").value = data["Team"];
            document.getElementById("name").value = data["Name"];
        },
        error: function(request, status, error) {
            alert("Shotgun에서 정보를 가져오는 동안 문제가 발생하였습니다.");
        }
    })
}

/* 타임로그 관련 함수 */
// sortTableFunc 함수는 입력받은 열을 오름차순으로 정렬하는 함수이다.
function sortTableFunc(tableID, col) {
    let table = document.getElementById(tableID);
    var switching = true;
    while (switching) {
        switching = false;
        for (var i = 1; i < table.rows.length - 2; i++) { // 마지막 total 행은 제외
            x = table.rows[i].getElementsByTagName("td")[col];
            y = table.rows[i + 1].getElementsByTagName("td")[col];
            if (col == 0) {
                if (tableID == "timelogtable-vfx") {
                    if (Number(x.innerHTML) > Number(y.innerHTML)) {
                        table.rows[i].parentNode.insertBefore(table.rows[i + 1], table.rows[i]);
                        switching = true;
                        break;
                    }
                }
                else {
                    if (x.innerHTML.toLowerCase() > y.innerHTML.toLowerCase()) {
                        table.rows[i].parentNode.insertBefore(table.rows[i + 1], table.rows[i]);
                        switching = true;
                        break;
                    }
                }
            }
            else {
                if (x.innerHTML.toLowerCase() > y.innerHTML.toLowerCase()) {
                    table.rows[i].parentNode.insertBefore(table.rows[i + 1], table.rows[i]);
                    switching = true;
                    break;
                }
            }
        }
    }

    // 클릭한 column이 sort되었다는 것을 보여주기 위하여 클래스 추가하고, 다른 column은 클래스를 삭제한다.
    if (col == 0) {
        document.getElementById("th0").classList.add("table-header-sorted")
        document.getElementById("th1").classList.remove("table-header-sorted")
    }
    else {
        document.getElementById("th0").classList.remove("table-header-sorted")
        document.getElementById("th1").classList.add("table-header-sorted")
    }
}

// sortTotalTableFunc 함수는 Total Timelog 페이지에서 vfx, cm을 나누어서 정렬하는 함수이다.
function sortTotalTableFunc(tableID, col) {
    let table = document.getElementById(tableID);
    var switching = true;
    while (switching) {
        switching = false;
        for (var i = 1; i < table.rows.length - 2; i++) { // 마지막 total 행은 제외
            x = table.rows[i].getElementsByTagName("td")[col];
            y = table.rows[i + 1].getElementsByTagName("td")[col];
            if (col == 1) {
                a = table.rows[i].getElementsByTagName("td")[0];
                b = table.rows[i + 1].getElementsByTagName("td")[0];

                if (x.innerHTML.toLowerCase() > y.innerHTML.toLowerCase()) {
                    if (!(a.innerHTML.indexOf('cm') == -1 && b.innerHTML.indexOf('cm') == 0)) {
                        table.rows[i].parentNode.insertBefore(table.rows[i + 1], table.rows[i]);
                        switching = true;
                        break;
                    }
                }
            }
            else {
                if (x.innerHTML.indexOf('cm') == -1 && y.innerHTML.indexOf('cm') == -1) {
                    if (Number(x.innerHTML) > Number(y.innerHTML)) {
                        table.rows[i].parentNode.insertBefore(table.rows[i + 1], table.rows[i]);
                        switching = true;
                        break;
                    }
                }
                else if (x.innerHTML.indexOf('cm') == 0 && y.innerHTML.indexOf('cm') == 0) {
                    if (x.innerHTML.toLowerCase() > y.innerHTML.toLowerCase()) {
                        table.rows[i].parentNode.insertBefore(table.rows[i + 1], table.rows[i]);
                        switching = true;
                        break;
                    }
                }
            }
        }
    }

    // 클릭한 column이 sort되었다는 것을 보여주기 위하여 클래스 추가하고, 다른 column은 클래스를 삭제한다.
    if (col == 0) {
        document.getElementById("th0").classList.add("table-header-sorted")
        document.getElementById("th1").classList.remove("table-header-sorted")
    }
    else {
        document.getElementById("th0").classList.remove("table-header-sorted")
        document.getElementById("th1").classList.add("table-header-sorted")
    }
}

// sortCheckTableFunc 함수는 Timelog Check 페이지에서 테이블을 이름 순으로 정렬하는 함수이다.
function sortCheckTableFunc(tableID) {
    let table = document.getElementById(tableID);
    var switching = true;
    while(switching) {
        switching = false;
        for (var i = 1; i < table.rows.length - 1; i++) { // 마지막 행도 포함
            x = table.rows[i].getElementsByTagName("td")[1];
            y = table.rows[i + 1].getElementsByTagName("td")[1];
            if (x.innerHTML.toLowerCase() > y.innerHTML.toLowerCase()) {
                table.rows[i].parentNode.insertBefore(table.rows[i + 1], table.rows[i]);
                switching = true;
                break;
            }
        }
    }
}

// changeVFXTeamComboFunc 함수는 VFX팀 타임로그 페이지의 검색바에서 부서를 선택했을 때 팀 콤보박스에 해당 부서의 팀만 보여주도록 수정해주는 함수이다.
function changeVFXTeamComboFunc(dept) {
    let token = document.getElementById("token").value;

    if (dept == "") {
        dept = "all"
    }
    $.ajax({
        url:`/api/vfxteams?dept=${dept}`,
        type: "get",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        async: false,
        success: function(data) {
            document.getElementById("team").length = 0; // 팀 콤보박스 초기화
            var option = document.createElement("option");
            option.appendChild(document.createTextNode("All"));
            option.value = "";
            document.getElementById("team").appendChild(option);
            for (var i = 0; i < data.length; i++) {
                var option = document.createElement("option");
                option.appendChild(document.createTextNode(data[i]));
                option.value = data[i];
                document.getElementById("team").appendChild(option);
            }
            document.getElementById("team").selectedIndex = 0;
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}\n\nAdmin setting에서 VFX 팀 정보를 가져오는 동안 문제가 발생하였습니다`);
        }
    })
}

//changeTotalTeamComboFunc 함수는 누계 타임로그 페이지의 검색바에서 부서를 선택했을 때 팀 콤보박스에 해당 부서의 팀만 보여주도록 수정해주는 함수이다.
function changeTotalTeamComboFunc(dept) {
    let token = document.getElementById("token").value;

    if (dept == "") {
        dept = "all"
    }
    $.ajax({
        url:`/api/totalteams?dept=${dept}`,
        type: "get",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        async:false,
        success: function(data) {
            document.getElementById("team").length = 0; // 팀 콤보박스 초기화
            var option = document.createElement("option");
            option.appendChild(document.createTextNode("All"));
            option.value = "";
            document.getElementById("team").appendChild(option);
            for (var i = 0; i < data.length; i++) {
                var option = document.createElement("option");
                option.appendChild(document.createTextNode(data[i]));
                option.value = data[i];
                document.getElementById("team").appendChild(option);
            }
            document.getElementById("team").selectedIndex = 0;
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}\n\nAdmin setting에서 Total 팀 정보를 가져오는 동안 문제가 발생하였습니다`);
        }
    })
}

// changeTimelogCMExcelURIFunc 함수는 CM팀 타임로그를 업데이트하는 페이지에서 날짜를 선택하면 실행되는 함수이다.
function changeTimelogCMExcelURIFunc() {
    let date = document.getElementById("date").value;
    document.getElementById("updateTimelogCMExcelURI").href = "/timelogcmexcel-submit?date=" + date;
}

// checkMonthlyStatusFunc 함수는 update 버튼을 눌렀을 때 월별 결산 상태에 맞게 modal창을 띄우는 함수이다.
function checkMonthlyStatusFunc() {
    let token = document.getElementById("token").value;

    $.ajax({
        url:`/api/checkmonthlystatus`,
        type: "post",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType:"json",
        async:false,
        success: function(data) {
           if (data.thismonth == true) {
               $("#modal-updatetimelog-thismonthtrue").modal("show");
           }
           else {
               if (data.lastmonth == true) {
                   $("#modal-updatetimelog-onlythismonth").modal("show");
                }
                else {
                    $("#modal-updatetimelog-lastmonthfalse").modal("show");
                }
            }
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}\n\n월별 결산 상태를 가져오지 동안 문제가 발생하였습니다`);
        }
    })
}

$("#modal-updatetimelog-onlythismonth").on("shown.bs.modal", function(){
    updateTimelogFunc(true);
});

$("#modal-updatetimelog-withlastmonth").on('shown.bs.modal', function(){
    updateTimelogFunc(false);
});

// updateTimelogFunc 함수는 타임로그를 업데이트하는 함수이다.
function updateTimelogFunc(status) {
    let token = document.getElementById("token").value;

    $.ajax({
        url:`/api/updatetimelog?status=${status}`,
        type: "post",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        success: function(data) {
            if (data.Status == true) {
                $("#modal-updatetimelog-onlythismonth").modal("hide");
            }
            else {
                $("#modal-updatetimelog-withlastmonth").modal("hide");
            }

            checkErrorTimelogFunc("noneprojects", data); // 예외 처리
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}\n\n타임로그를 업데이트하는 동안 문제가 발생하였습니다`);
        }
    })
}

// checkErrorTimelogFunc 함수는 타임로그 업데이트시 발생하는 예외들을 처리하는 함수이다.
function checkErrorTimelogFunc(errorType, data) {
    switch(errorType){
        case "noneprojects":
            if (data.Project) { // DB에 없는 프로젝트가 있으면 모달창을 띄운다.
                document.getElementById("modal-noneprojects-id").value = data.Project;
                $("#modal-noneprojects").modal("show");
                $("#modal-noneprojects").on("hide.bs.modal", function() { // 모달창이 닫히면 정산 완료된 프로젝트에 작성한 타임로그가 있는지 확인한다.
                    checkErrorTimelogFunc("finishedtimelog", data);
                })
            } else { // DB에 없는 프로젝트가 없으면 정산 완료된 프로젝트에 작성한 타임로그가 있는지 확인한다.
                checkErrorTimelogFunc("finishedtimelog", data)
            }
            break
        case "finishedtimelog":
            if (data.Timelog) { // 정산 완료된 프로젝트에 작성한 타임로그가 있는 경우
                if (data.InvalidAccess) { // admin 권한보다 낮으면 모달창을 띄운다.
                    $("#modal-finishedtimelog").modal("show");
                } else { // admin 권한이면 /finishedtimelog 로 리다이렉트한다.
                    window.localStorage.setItem(`finishedTimelognum`, data.Timelog.length); // Local Storage에 타임로그 개수 저장
                    for (var i = 0; i < data.Timelog.length; i++) {
                        window.localStorage.setItem(`finishedTimelog${i}`, JSON.stringify(data.Timelog[i])); // Local Storage에 타임로그 정보 저장
                    }
                    window.location.href = `/finishedtimelog?status=${data.Status}`;
                }
            } else {
                window.location.reload();  // 페이지 새로고침
            }
            break
    }
}

// setTableOfFinishedTimglogFunc 함수는 finished timelog 페이지를 띄웠을 때 실행되는 함수이다.
function setTableOfFinishedTimglogFunc(status) {
    // Local Storage에 저장된 타임로그 정보를 가져온다.
    var now = new Date();
    var year = now.getFullYear();
    var month = now.getMonth() + 1;
    var totalTimelogNum = window.localStorage.getItem("finishedTimelognum");
    var timelogList = new Map(); // 이번달의 타임로그 리스트
    var timelogNum = 0;
    var lastTimelogList = new Map(); // 지난달의 타임로그 리스트
    var lastTimelogNum = 0;
    for (var i = 0; i < totalTimelogNum; i++) { 
        var timelog = JSON.parse(window.localStorage.getItem(`finishedTimelog${i}`));
        if (timelog.year == year && timelog.month == month) { // 타임로그가 이번달에 작성했을 경우 timelogList에 추가
            if (timelogList.has(timelog.project)) {
                var value = timelogList.get(timelog.project);
                value.push(timelog);
                timelogList[timelog.project] = value;
            } else{
                timelogList.set(timelog.project, [timelog]);
            }
            timelogNum = timelogNum + 1;
        } else { // 타임로그가 지난달에 작성했을 경우 lastTimelogList에 추가
            if (lastTimelogList.has(timelog.project)) {
                var value = lastTimelogList.get(timelog.project);
                value.push(timelog);
                lastTimelogList[timelog.project] = value;
            } else{
                lastTimelogList.set(timelog.project, [timelog]);
            }
            lastTimelogNum = lastTimelogNum + 1;
        }
    }

    // 테이블에 이번달에 작성한 타임로그를 추가한다.
    document.getElementById("timelognum").value = timelogNum;
    var mapIterator = timelogList.keys();
    var index = 0
    var tableBody = document.getElementById("nowtable");
    for (let project of mapIterator) {
        var timelogs = timelogList.get(project);
        
        for (var i = 0; i < timelogs.length; i++) {
            var row = tableBody.insertRow(tableBody.rows.length);
            var duration = (timelogs[i].duration / 60.0).toFixed(1);
            var rowData = "";
            if (i == 0) {
                rowData += `
                <tr>
                    <td rowspan=${timelogs.length}>${project}</td>
                    <td>${timelogs[i].userid}</td>
                    <td>${duration}h</td>
                    <td rowspan=${timelogs.length}>
                        <div class="custom-control custom-checkbox custom-control-inline">
                            <input type="checkbox" class="custom-control-input" name="projectstatus${index}" id="projectstatus${index}" checked onclick="setRndCheckBoxFunc('projectstatus${index}', 'etcstatus${index}')">
                            <label class="text-white custom-control-label" for="projectstatus${index}">프로젝트로 처리</label>
                        </div>
                        <div class="custom-control custom-checkbox custom-control-inline">
                            <input type="checkbox" class="custom-control-input" name="etcstatus${index}" id="etcstatus${index}" onclick="setRndCheckBoxFunc('etcstatus${index}', 'projectstatus${index}')">
                            <label class="text-white custom-control-label" for="etcstatus${index}">ETC로 처리</label>
                        </div>
                    </td>
                    <input type="hidden" id="project${index}" name="project${index}" value="${project}">
                    <input type="hidden" id="userid${index}" name="userid${index}" value="${timelogs[i].userid}">
                </tr>
                `;
            } else {
                index = index + 1;
                rowData += `
                <tr>
                    <td>${timelogs[i].userid}</td>
                    <td>${duration}h</td>
                    <input type="hidden" id="project${index}" name="project${index}" value="${project}">
                    <input type="hidden" id="userid${index}" name="userid${index}" value="${timelogs[i].userid}">
                </tr>
                `;
            }
            row.innerHTML = rowData;
            tableBody.append(row);
        }
        index = index + 1;
    }

    // 지난달까지 업데이트했을 경우 테이블에 지난달에 작성한 타임로그를 추가한다.
    if (status == "false") {
        document.getElementById("lasttimelognum").value = lastTimelogNum;
        mapIterator = lastTimelogList.keys();
        index = 0
        tableBody = document.getElementById("lasttable");
        for (let project of mapIterator) {
            var timelogs = lastTimelogList.get(project);
        
            for (var i = 0; i < timelogs.length; i++) {
                var row = tableBody.insertRow(tableBody.rows.length);
                var duration = (timelogs[i].duration / 60.0).toFixed(1);
                var rowData = "";
                if (i == 0) {
                    rowData += `
                    <tr>
                        <td rowspan=${timelogs.length}>${project}</td>
                        <td>${timelogs[i].userid}</td>
                        <td>${duration}h</td>
                        <td rowspan=${timelogs.length}>
                            <div class="custom-control custom-checkbox custom-control-inline">
                                <input type="checkbox" class="custom-control-input" name="lastprojectstatus${index}" id="lastprojectstatus${index}" checked onclick="setRndCheckBoxFunc('lastprojectstatus${index}', 'lastetcstatus${index}')">
                                <label class="text-white custom-control-label" for="lastprojectstatus${index}">프로젝트로 처리</label>
                            </div>
                            <div class="custom-control custom-checkbox custom-control-inline">
                                <input type="checkbox" class="custom-control-input" name="lastetcstatus${index}" id="lastetcstatus${index}" onclick="setRndCheckBoxFunc('lastetcstatus${index}', 'lastprojectstatus${index}')">
                                <label class="text-white custom-control-label" for="lastetcstatus${index}">ETC로 처리</label>
                            </div>
                        </td>
                        <input type="hidden" id="lastproject${index}" name="lastproject${index}" value="${project}">
                        <input type="hidden" id="lastuserid${index}" name="lastuserid${index}" value="${timelogs[i].userid}">
                    </tr>
                    `;
                } else {
                    index = index + 1;
                    rowData += `
                    <tr>
                        <td>${timelogs[i].userid}</td>
                        <td>${duration}h</td>
                        <input type="hidden" id="lastproject${index}" name="lastproject${index}" value="${project}">
                        <input type="hidden" id="lastuserid${index}" name="lastuserid${index}" value="${timelogs[i].userid}">
                    </tr>
                    `;
                }
                row.innerHTML = rowData;
                tableBody.append(row);
            }
            index = index + 1;
        }
    }
}

function checkFinishedTimelogFunc(status) {
    // 이번달의 타임로그 테이블에 체크를 모두 했는지 체크한다.
    var timelogNum = document.getElementById("timelognum").value;
    for (var i = 1; i < timelogNum; i++) {
        var projectStatus = document.getElementById(`projectstatus${i}`);
        if (!projectStatus) { // 프로젝트별로 체크박스가 하나로 통일되어 있기 때문에 이 ID를 가진 체크박스가 없을 수도 있다.
            continue
        }
        var etcStatus = document.getElementById(`etcstatus${i}`);
        if (!etcStatus) {
            continue
        }
        if (projectStatus.checked == false && etcStatus.checked == false) {
            alert("프로젝트로 처리 혹은 ETC로 처리 옵션 중 하나를 선택해주세요")
            return false
        }
    }

    if (!status) {
        // 지난달의 타임로그 테이블에 체크를 모두 했는지 체크한다.
        timelogNum = document.getElementById("lasttimelognum").value;
        for (var i = 0; i < timelogNum; i++) {
            var projectStatus = document.getElementById(`lastprojectstatus${i}`);
            if (!projectStatus) {
                continue
            }
            var etcStatus = document.getElementById(`lastetcstatus${i}`);
            if (!etcStatus) {
                continue
            }
            if (projectStatus.checked == false && etcStatus.checked == false) {
                alert("프로젝트로 처리 혹은 ETC로 처리 옵션 중 하나를 선택해주세요")
                return false
            }
        }
    }

    // Local Storage에 저장된 타임로그 정보를 삭제한다.
    var totalTimelogNum = window.localStorage.getItem("finishedTimelognum");
    window.localStorage.removeItem(`finishedTimelognum`);
    for (var i = 0; i < totalTimelogNum; i++) {
        window.localStorage.removeItem(`finishedTimelog${i}`);
    }
    return true
}

// setRmTimelogByIDModalFunc 함수는 제외할 ID의 타임로그를 삭제하는 버튼을 클릭하면 ID를 받아 modal 창에 보여주는 함수이다.
function setRmTimelogByIDModalFunc(idList) {
    document.getElementById("modal-rmtimelogbyid-id").value = idList;
}

// rmTimelogByIDFunc 함수는 restAPI를 이용하여 입력받은 ID가 작성한 타임로그를 삭제하는 함수이다.
function rmTimelogByIDFunc(idList) {
    let token = document.getElementById("token").value;

    $.ajax({
        url:`/api/rmtimelogbyid?id=${idList}`,
        type: "delete",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        success: function() {
            alert("삭제되었습니다")
            $("#modal-rmtimelogbyid").modal("hide")
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}\n\n입력한 ID의 아티스트가 작성한 타임로그를 삭제하는 동안 문제가 발생하였습니다`);
        }
    })
}

// setRmTimelogByProjectModalFunc 함수는 제외할 프로젝트의 타임로그를 삭제하는 버튼을 클릭하면 프로젝트를 받아 modal 창에 보여주는 함수이다.
function setRmTimelogByProjectModalFunc(projectList) {
    document.getElementById("modal-rmtimelogbyproject-project").value = projectList;
}

// rmTimelogByProjectFunc 함수는 restAPI를 이용하여 입력받은 프로젝트에 작성한 타임로그를 삭제하는 함수이다.
function rmTimelogByProjectFunc(projectList) {
    let token = document.getElementById("token").value;

    $.ajax({
        url:`/api/rmtimelogbyproject?project=${projectList}`,
        type: "delete",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        success: function() {
            alert("삭제되었습니다")
            $("#modal-rmtimelogbyproject").modal("hide")
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}\n\n입력한 프로젝트에 작성한 타임로그를 삭제하는 동안 문제가 발생하였습니다`);
        }
    })
}

// reset timelog modal이 실행되면 resetTimelogFunc() 함수를 실행한다.
$("#modal-resettimelog").on("shown.bs.modal", function(){
    resetTimelogFunc();
});

// resetTimelogFunc 함수는 restAPI를 이용하여 타임로그를 리셋하는 함수이다.
function resetTimelogFunc() {
    let token = document.getElementById("token").value;

    $.ajax({
        url:`/api/resettimelog`,
        type: "post",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        success: function() {
            alert("리셋되었습니다")
            $("#modal-resettimelog").modal("hide")
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}\n\n타임로그를 리셋하는 동안 문제가 발생하였습니다`);
        }
    })
}

// calTotalSupTimelogFunc 함수는 수퍼바이저의 타임로그를 입력했을 때 total을 계산하여 보여주는 함수이다.
function calTotalSupTimelogFunc(rowNum) {
    var tr = document.getElementById("suptimelogstable").rows[Number(rowNum) + 2];

    var totalDuration = 0.0;
    for (var i = 2; i < tr.cells.length - 1; i++) {
        duration = tr.cells[i].querySelector("input").value;
        if (duration != "") {
            totalDuration += Number(duration);
        }
    }

    tr.cells[tr.cells.length - 1].innerHTML = totalDuration + "h" // total의 텍스트를 바꿔준다.
}

/* 프로젝트 관련 함수 */
// checkIsFinishedInAddFunc 함수는 add project에서 isfinished가 체크되었는지 확인하는 함수이다.
function checkIsFinishedInAddFunc() {
    var checkbox = document.getElementById("isfinished");
    var totalamount = document.getElementById("totalamount");
    var laborcost = document.getElementById("laborcost");
    var progresscost = document.getElementById("progresscost");
    var purchasecost = document.getElementById("purchasecost");
    var difference = document.getElementById("difference");

    if (checkbox.checked == true) {
        // 정산 완료된 프로젝트를 체크하는 경우
        totalamount.disabled = false
        laborcost.disabled = false
        progresscost.disabled = false
        purchasecost.disabled = false
        difference.disabled = false
    }
    else {
        // 정산 완료된 프로젝트 체크박스의 체크를 해제하는 경우
        totalamount.value = ""
        laborcost.value = ""
        progresscost.value = ""
        purchasecost.value = ""
        difference.value = ""
        totalamount.disabled = true
        laborcost.disabled = true
        progresscost.disabled = true
        purchasecost.disabled = true
        difference.disabled = true
    }
}

// checkIsFinishedFunc 함수는 edit project에서 isfinished가 체크되었는지 확인하는 함수이다.
function checkIsFinishedInEditFunc() {
    var checkbox = document.getElementById("isfinished");
    var radio1 = document.getElementById("typeCheckbox1");
    var radio2 = document.getElementById("typeCheckbox2");
    var totalamount = document.getElementById("totalamount");
    var laborcost = document.getElementById("laborcost");
    var progresscost = document.getElementById("progresscost");
    var purchasecost = document.getElementById("purchasecost");
    var difference = document.getElementById("difference");

    if (checkbox.checked == true && radio1.checked == false && radio2.checked == false) {
        // 처음에 정산 완료된 프로젝트 체크박스를 체크하는 경우, 이때는 radio 버튼 모두 비활성화되어있다.
        radio1.checked = true
        radio1.disabled = false
        radio2.disabled = false
        difference.disabled = false
    }
    else if (checkbox.checked == true && radio2.checked == true) {
        // 정산 완료된 프로젝트 체크박스가 체크되어 있고, 입력값으로 저장 radio 버튼을 눌렀을 경우
        totalamount.disabled = false
        laborcost.disabled = false
        progresscost.disabled = false
        purchasecost.disabled = false
    }
    else if (checkbox.checked == true && radio1.checked == true) {
        // 정산 완료된 프로젝트 체크박스가 체크되어 있고, 계산된 값으로 저장 radio 버튼을 눌렀을 경우
        totalamount.value = ""
        laborcost.value = ""
        progresscost.value = ""
        purchasecost.value = ""
        totalamount.disabled = true
        laborcost.disabled = true
        progresscost.disabled = true
        purchasecost.disabled = true
    }
    else {
        // 정산 완료된 프로젝트 체크박스를 해제하는 경우
        radio1.checked = false
        radio2.checked = false
        radio1.disabled = true
        radio2.disabled = true
        totalamount.value = ""
        laborcost.value = ""
        progresscost.value = ""
        purchasecost.value = ""
        difference.value = ""
        totalamount.disabled = true
        laborcost.disabled = true
        progresscost.disabled = true
        purchasecost.disabled = true
        difference.disabled = true
    }
}

// setIsFinishedFunc 함수는 edit project에서 초기에 isfinished 부분을 체크하는 함수이다.
function setIsFinishedInEditFunc(typ) {
    var checkbox = document.getElementById("isfinished");
    var radio1 = document.getElementById("typeCheckbox1");
    var radio2 = document.getElementById("typeCheckbox2");
    var totalamount = document.getElementById("totalamount");
    var laborcost = document.getElementById("laborcost");
    var progresscost = document.getElementById("progresscost");
    var purchasecost = document.getElementById("purchasecost");
    var difference = document.getElementById("difference");

    if (checkbox.checked == true) { // Edit 페이지가 처음에 띄워질 때 정산 완료된 프로젝트 체크박스가 체크되어있는 경우
        if (typ == "true") { // 월별 합산 값으로 저장할 경우
            radio1.disabled = false
            radio2.disabled = false
            radio1.checked = true
            totalamount.disabled = true
            laborcost.disabled = true
            progresscost.disabled = true
            purchasecost.disabled = true
        } else { // 최종 입력 값으로 저장할 경우
            radio1.disabled = false
            radio2.disabled = false
            radio2.checked = true
            totalamount.disabled = false
            laborcost.disabled = false
            progresscost.disabled = false
            purchasecost.disabled = false
        }
    }
}

// addProjectPageBlankCheckFunc 함수는 프로젝트 추가, 수정 페이지에서 빈칸이 있는지 확인하는 함수이다.
function addProjectPageBlankCheckFunc(type) {
    // project add -> true, project edit -> false
    if (document.getElementById("id").value == "") {
        alert("Project ID를 입력해주세요");
        return false;
    }
    if (document.getElementById("name").value == "") {
        alert("프로젝트 한글명을 입력해주세요");
        return false;
    }
    if (document.getElementById("startdate").value == "") {
        alert("작업 시작일을 입력해주세요");
        return false;
    }
    if (document.getElementById("enddate").value == "") {
        alert("작업 마감일를 입력해주세요");
        return false;
    }
    var startdate = new Date(document.getElementById("startdate").value)
    var enddate = new Date(document.getElementById("enddate").value)
    if (startdate > enddate) {
        alert("작업 시작일과 작업 마감일이 잘못 입력되었습니다.\n다시 확인해주세요.")
        return false;
    }
    if (document.getElementById("payment0").value == "") {
        alert("총 매출을 입력해주세요");
        return false;
    }
    if (document.getElementById("paymentdate0").value == "") {
        alert("계약일을 입력해주세요");
        return false;
    }

    if (type == false) { // 프로젝트 수정 페이지일 때 아래의 사항들을 체크한다.
        var checkbox = document.getElementById("isfinished");
        var radio1 = document.getElementById("typeCheckbox1");

        // 총 매출과 계약일 모두 입력했는지 확인한다.
        var totalPayment = 0;
        var paymentNum = document.getElementById("paymentNum").value;
        for (var i = 0; i < paymentNum; i++) {
            var payment = document.getElementById(`payment${i}`).value;
            var paymentDate = document.getElementById(`paymentdate${i}`).value;
            if (payment == "" && paymentDate == "") {
                continue
            } else if (payment == "" || paymentDate == "") {
                alert("총 매출과 계약일 모두 입력해주세요");
                return false;
            }

            if (payment.includes(",")) {
                payment = payment.replace(/,/gi, "");
            }
            totalPayment += Number(payment);
        }

        // 총 매출과 월별 매출의 합을 비교한다.
        var summonthlypayment = 0
        while (true) {
            var year = startdate.getFullYear().toString()
            var tempdate = new Date(startdate.getUTCFullYear(), startdate.getMonth() + 1) // 자바스크립트에서는 월이 0부터 시작한다. 
            var month = tempdate.getMonth().toString()
            if (tempdate.getMonth() == 0) {
                month = "12"
            }
            var sd = year + "-" + month.padStart(2, "0") + "monthlypayment"
            if (document.getElementById(sd) != null) { // 작업기간을 수정한 경우 존재하지 않는 칸의 value를 가져올 때 콘솔 로그에 에러메시지가 나온다.
                let monthlypayment = document.getElementById(sd).value;
                if (monthlypayment.includes(",")) {
                    monthlypayment = monthlypayment.replace(/,/gi, '')
                }
                let monthlypaymentNum = Number(monthlypayment)
                summonthlypayment = summonthlypayment + monthlypaymentNum
            }
            if (startdate.getFullYear()== enddate.getFullYear() && startdate.getMonth() == enddate.getMonth()) {
                break
            }
            startdate.setMonth(startdate.getMonth() + 1)
        }
        if (checkbox.checked == true && radio1.checked == true) { // 월별 합산 값으로 정산할 경우 총 매출과 월별 매출의 합이 같은지 확인한다.
            if (totalPayment != summonthlypayment) {
                alert("총 매출과 월별 매출의 합이 다릅니다.\n다시 확인해주세요.")
                return false
            }
        } else if (checkbox.checked == false) { // 정산 완료된 프로젝트를 체크하지 않았을 경우 월별 매출의 합이 총 매출보다 작은지 확인한다.
            if (totalPayment < summonthlypayment) {
                alert("월별 매출의 합이 총 매출보다 큽니다.\n다시 확인해주세요.")
                return false
            }
        }

        // 최종 입력 값으로 정산할 경우
        if (checkbox.checked == true && radio1.checked == false) {
            let totalAmount = document.getElementById("totalamount").value;
            if (totalAmount.includes(",")) {
                totalAmount = totalAmount.replace(/,/gi, '')
            }
            let totalAmountNum = Number(totalAmount);

            // 총 내부비용을 입력했는지 확인한다.
            if (totalAmount == "") {
                alert("총 내부비용을 입력해주세요.")
                return false
            }

            // 총 내부비용과 내부인건비, 진행비, 구매비의 합이 같은지 확인한다.
            let laborCost = document.getElementById("laborcost").value;
            if (laborCost.includes(",")) {
                laborCost = laborCost.replace(/,/gi, '')
            }
            let laborCostNum = Number(laborCost);
            let progressCost = document.getElementById("progresscost").value;
            if (progressCost.includes(",")) {
                progressCost = progressCost.replace(/,/gi, '')
            }
            let progressCostNum = Number(progressCost);
            let purchaseCost = document.getElementById("purchasecost").value;
            if (purchaseCost.includes(",")) {
                purchaseCost = purchaseCost.replace(/,/gi, '')
            }
            let purchaseCostNum = Number(purchaseCost);
            var total = laborCostNum + progressCostNum + purchaseCostNum;
            if (laborCost != "" || progressCost != "" || purchaseCost != "") {
                if (totalAmountNum != total) {
                    alert("총 내부비용과 내부인건비,진행비,구매비의 합이 다릅니다.\n다시 확인해주세요.")
                    return false
                }
            }
            if (!confirm("최종 입력 값으로 저장할 경우 월별로 입력한 모든 데이터가 삭제됩니다. 진행하시겠습니까?")) {
                return false
            }
        }
    } else { // 프로젝트 추가 페이지일 때 아래의 사항들을 체크한다.
        var checkbox = document.getElementById("isfinished");
        if (checkbox.checked == true) {
            let totalAmount = document.getElementById("totalamount").value;
            if (totalAmount.includes(",")) {
                totalAmount = totalAmount.replace(/,/gi, '');
            }
            let totalAmountNum = Number(totalAmount);

            // 총 내부비용을 입력했는지 확인한다.
            if (totalAmount == "") {
                alert("총 내부비용을 입력해주세요.")
                return false
            }

            // 총 내부비용과 내부인건비, 진행비, 구매비의 합이 같은지 확인한다.
            let laborCost = document.getElementById("laborcost").value;
            if (laborCost.includes(",")) {
                laborCost = laborCost.replace(/,/gi, '');
            }
            let laborCostNum = Number(laborCost);
            let progressCost = document.getElementById("progresscost").value;
            if (progressCost.includes(",")) {
                progressCost = progressCost.replace(/,/gi, '');
            }
            let progressCostNum = Number(progressCost)
            let purchaseCost = document.getElementById("purchasecost").value;
            if (purchaseCost.includes(",")) {
                purchaseCost = purchaseCost.replace(/,/gi, '');
            }
            let purchaseCostNum = Number(purchaseCost)
            let total = laborCostNum + progressCostNum + purchaseCostNum;

            if (laborCost != "" || progressCost != "" || purchaseCost != "") {
                if (totalAmountNum != total) {
                    alert("총 내부비용과 내부인건비,진행비,구매비의 합이 다릅니다.\n다시 확인해주세요.")
                    return false
                }
            }
        }
    }
}

// setStatusCheckBoxFunc 함수는 status 버튼을 클릭했을 때 체크박스가 체크/해제되도록 하고, status에 추가/삭제하는 함수이다.
function setStatusCheckBoxFunc(id) {
    document.getElementById(id).checked = !document.getElementById(id).checked

    var trueStatus = document.getElementById("status").value;
    var checked = document.getElementById(id).checked;
    var status = document.getElementById(id).value;

    if (trueStatus == "" && checked) {
        document.getElementById("status").value = status
        return
    }

    var trueStatusList = trueStatus.split(",");
    if (checked) {  // 체크박스가 체크되었을 때
        if (!trueStatusList.includes(status)) {
            trueStatusList.splice(0, 0, status);
        }
    } else {  // 체크박스가 해제되었을 때
        var index = trueStatusList.indexOf(status);
        if (index > -1) {
            trueStatusList.splice(index, 1);
        }
    }

    document.getElementById("status").value = trueStatusList.join(",");
}

// setRmProjectModalFunc 함수는 프로젝트 삭제 버튼을 클릭하면 ID, 이름을 받아 modal 창에 보여주는 함수이다.
function setRmProjectModalFunc(id, name) {
    document.getElementById("modal-rmproject-id").value = id;
    document.getElementById("modal-rmproject-name").value = name;
}

// rmProjectFunc 함수는 restAPI를 이용하여 프로젝트를 삭제하는 함수이다.
function rmProjectFunc(id) {
    let token = document.getElementById("token").value;

    $.ajax({
        url: `/api/rmproject?id=${id}`,
        type: "delete",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        success: function(data) {
            alert("프로젝트가 삭제되었습니다.")
            location.reload();  // 페이지 새로고침
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}`);
        }
    })
}

// setMonthlyPurchaseCostModalFunc 함수는 ... 버튼을 클릭하면 프로젝트 ID, 날짜, 구매 내역을 modal 창에 보여주는 함수이다.
function setMonthlyPurchaseCostModalFunc(projectID, date) {
    let token = document.getElementById("token").value;
    
    $.ajax({
        url: `/api/monthlyPurchaseCost?id=${projectID}&date=${date}`,
        type: "get",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        success: function(data) {
            let parent = document.getElementById("purchaseCost");
            while (parent.firstChild) {  // 입력칸들을 모두 삭제한다.
                parent.removeChild(parent.firstChild);
            }

            var year = date.split("-")[0];
            var month = date.split("-")[1];
            document.getElementById("modal-setMonthlyPurchaseCost-title").innerHTML = `${year}년 ${month}월 구매 내역`;
            document.getElementById("projectID").value = projectID;
            document.getElementById("date").value = date;

            if (!data) {  // 구매 내역이 없으면 빈 칸을 추가한다.
                var e = document.createElement("div");
                e.className = "form-row";
                html = `
                <div class="col pt-2">
		            <input type="text" class="form-control" id="companyName0" name="companyName0" placeholder="업체명">
		        </div>
		        <div class="col-7 pt-2">
		            <input type="text" class="form-control" id="detail0" name="detail0" placeholder="내역">
		        </div>
		        <div class="col pt-2">
		            <input type="text" inputmode="numeric" class="form-control" id="expenses0" name="expenses0" placeholder="금액">
			    </div>
                `
                e.innerHTML = html;
                parent.appendChild(e);
            } else {
                for (var i = 0; i < data.length; i++) {
                    var companyName = data[i].CompanyName;
                    var detail = data[i].Detail;
                    var expensesNum = Number(data[i].Expenses);
                    let expenses = expensesNum.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",")
                    var e = document.createElement("div");
                    e.className = "form-row";
                    html = `
                    <div class="col pt-2">
		                <input type="text" class="form-control" id="companyName${i}" name="companyName${i}" placeholder="업체명" value="${companyName}">
		            </div>
		            <div class="col-7 pt-2">
		                <input type="text" class="form-control" id="detail${i}" name="detail${i}" placeholder="내역" value="${detail}">
		            </div>
		            <div class="col pt-2">
		                <input type="text" inputmode="numeric" class="form-control" id="expenses${i}" name="expenses${i}" placeholder="금액" value="${expenses}">
		    	    </div>
                    `
                    e.innerHTML = html;
                    parent.appendChild(e);
                }
            }
            document.getElementById("purchaseCostNum").value = parent.childElementCount; // 최종 생성된 구매 내역 개수를 purchaseCostNum에 저장한다.
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}`);
        }
    })
}

// addPurchaseCostFunc 함수는 + 버튼을 클릭하면 구매 내역을 입력할 수 있는 칸을 하나 더 늘려주는 함수이다.
function addPurchaseCostFunc() {
    let childNum = document.getElementById("purchaseCost").childElementCount;
    let e = document.createElement("div");
    e.className = "form-row";
    let html = `
    <div class="col pt-2">
        <input type="text" class="form-control" id="companyName${childNum}" name="companyName${childNum}" placeholder="업체명">
    </div>
    <div class="col-7 pt-2">
        <input type="text" class="form-control" id="detail${childNum}" name="detail${childNum}" placeholder="내역">
    </div>
    <div class="col pt-2">
        <input type="text" inputmode="numeric" class="form-control" id="expenses${childNum}" name="expenses${childNum}" placeholder="금액">
    </div>
    `
    e.innerHTML = html;
    document.getElementById("purchaseCost").appendChild(e);
    document.getElementById("purchaseCostNum").value = document.getElementById("purchaseCost").childElementCount; // 최종 생성된 구매 내역 개수를 purchaseCostNum에 저장한다.
}

// setMonthlyPurchaseCostFunc 함수는 modal-setMonthlyPurchaseCost에서 Update 버튼을 클릭하면 restAPI를 이용하여 월별 구매 내역을 업데이트하는 함수이다.
function setMonthlyPurchaseCostFunc() {
    let token = document.getElementById("token").value;
    let id = document.getElementById("projectID").value;
    let date = document.getElementById("date").value;
    let num = document.getElementById("purchaseCostNum").value;

    var url = []
    for (var i = 0; i < num; i++) {
        var companyName = document.getElementById(`companyName${i}`).value;
        var detail = document.getElementById(`detail${i}`).value;
        var expenses = document.getElementById(`expenses${i}`).value;
        if (expenses.includes(",")) {
            expenses = expenses.replace(/,/gi, '');
        }
        if (companyName == "" && detail == "" && expenses == "") {  // 업체명, 내역, 금액 모두 빈칸이면 continue
            continue
        }
        if (companyName == "" && detail == "") {
            alert("업체명, 내역 중 하나는 필수로 입력해주세요.")
            return
        }
        if (expenses == "") {
            alert("금액을 입력해주세요")
            return
        }
        url.push(`companyName${i}=${companyName}&detail${i}=${detail}&expenses${i}=${expenses}`)
    }
    url.push(`num=${num}`)

    $.ajax({
        url: `/api/setMonthlyPurchaseCost?id=${id}&date=${date}&${url.join("&")}`,
        type: "post",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        success: function(data) {
            alert("구매 내역이 저장되었습니다.");
            $("#modal-setMonthlyPurchaseCost").modal("hide");
            if (data == "0") {  // 총합이 0이면 빈 문자열을 보여준다.
                data = "";
            }
            let expenses = data.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",")
            document.getElementById(`${date}smpurchasecost`).value = expenses;  // 총액
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}`);
        }
    })
}

// addPaymentFunc 함수는 총 매출 추가 버튼을 클릭하면 총 매출을 입력할 수 있는 칸을 하나 더 늘려주는 함수이다.
function addPaymentFunc() {
    let childNum = document.getElementById("addPayment").childElementCount;
    let e = document.createElement("div");
    let html = `
    <div class="row">
        <div class="col">
            <div class="form-group pb-2">
                <label class="text-muted">총 매출 ${childNum + 1}</label>
                <input type="text" inputmode="numeric" class="form-control" id="payment${childNum}" name="payment${childNum}" value="">
                <small class="form-text text-muted">숫자만 입력해주세요.</small>
            </div>
        </div>
        <div class="col">
            <div class="form-group pb-2">
                <label class="text-muted">계약일 ${childNum + 1}</label>
                <input type="date" class="form-control" id="paymentdate${childNum}" name="paymentdate${childNum}" value="" max="9999-12-31">
            </div>
        </div>
    </div>
    `
    e.innerHTML = html;
    document.getElementById("addPayment").appendChild(e);
    document.getElementById("paymentNum").value = document.getElementById("addPayment").childElementCount; // 최종 생성된 매출 내역 개수를 paymentNum에 저장한다.
}

// setMonthlyPaymentModalFunc 함수는 ... 버튼을 클릭하면 프로젝트 ID, 날짜, 매출 내역을 modal 창에 보여주는 함수이다.
function setMonthlyPaymentModalFunc(projectID, date) {
    let token = document.getElementById("token").value;
    
    $.ajax({
        url: `/api/monthlyPayment?id=${projectID}&date=${date}`,
        type: "get",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        success: function(data) {
            let parent = document.getElementById("modal-setMonthlyPayment-payment");
            while (parent.firstChild) {  // 입력칸들을 모두 삭제한다.
                parent.removeChild(parent.firstChild);
            }

            var year = date.split("-")[0];
            var month = date.split("-")[1];
            document.getElementById("modal-setMonthlyPayment-title").innerHTML = `${year}년 ${month}월 매출 내역`;
            document.getElementById("modal-setMonthlyPayment-projectID").value = projectID;
            document.getElementById("modal-setMonthlyPayment-date").value = date;

            if (!data) {  // 매출 내역이 없으면 빈 칸을 추가한다.
                var e = document.createElement("div");
                e.className = "form-row align-items-center";
                html = `
                <div class="col">
                    <select id="modal-setMonthlyPayment-type0" name="modal-setMonthlyPayment-type0" class="form-control">
                        <option value=""></option>
                        <option value="계약금">계약금</option>
                        <option value="중도금">중도금</option>
                        <option value="잔금">잔금</option>
                    </select>
                </div>
                <div class="col">
		            <input type="text" inputmode="numeric" class="form-control" id="modal-setMonthlyPayment-expenses0" name="modal-setMonthlyPayment-expenses0" placeholder="금액">
                </div>
                <div class="col">
		            <input type="date" class="form-control" id="modal-setMonthlyPayment-date0" name="modal-setMonthlyPayment-date0" max="9999-12-31">
		        </div>
                <div class="col">
		            <input type="date" class="form-control" id="modal-setMonthlyPayment-depositdate0" name="modal-setMonthlyPayment-depositdate0" max="9999-12-31">
		        </div>
                <div class="col text-right">
                    <div class="custom-control custom-radio custom-control-inline">
                        <input type="radio" id="modal-setMonthlyPayment-statusone0" name="modal-setMonthlyPayment-status0" class="custom-control-input" value="true">
                        <label class="custom-control-label text-muted" for="modal-setMonthlyPayment-statusone0">Yes</label>
                    </div>
                    <div class="custom-control custom-radio custom-control-inline">
                        <input type="radio" id="modal-setMonthlyPayment-statustwo0" name="modal-setMonthlyPayment-status0" class="custom-control-input" value="false" checked>
                        <label class="custom-control-label text-muted" for="modal-setMonthlyPayment-statustwo0">No</label>
                    </div>
                </div>
                <div class="col-1">
                    <span class="badge badge-pill badge-danger text-center finger" onclick="delMonthlyPaymentFunc(0)">Del</span>
                </div>
                `
                e.innerHTML = html;
                parent.appendChild(e);
            } else {
                for (var i = 0; i < data.length; i++) {
                    var typeChecked1 = "";
                    var typeChecked2 = "";
                    var typeChecked3 = "";
                    var typeChecked4 = "";
                    switch (data[i].Type) {
                        case "계약금":
                            typeChecked2 = "selected";
                            break;
                        case "중도금":
                            typeChecked3 = "selected";
                            break;
                        case "잔금":
                            typeChecked4 = "selected";
                            break;
                        default:
                            typeChecked1 = "selected";
                            break;
                    }
                    var expensesNum = Number(data[i].Expenses);
                    let expenses = expensesNum.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
                    var status1 = "";
                    var status2 = "";
                    if (data[i].Status) {
                        status1 = "checked";
                    } else {
                        status2 = "checked";
                    }
                    var pt = "pt-2";
                    if (i == 0) {
                        pt = "";
                    }

                    var e = document.createElement("div");
                    e.className = "form-row align-items-center";
                    html = `
                    <div class="col ${pt}">
                        <select id="modal-setMonthlyPayment-type${i}" name="modal-setMonthlyPayment-type${i}" class="form-control">
                            <option value="" ${typeChecked1}></option>
                            <option value="계약금" ${typeChecked2}>계약금</option>
                            <option value="중도금" ${typeChecked3}>중도금</option>
                            <option value="잔금" ${typeChecked4}>잔금</option>
                        </select>
                    </div>
                    <div class="col ${pt}">
		                <input type="text" inputmode="numeric" class="form-control" id="modal-setMonthlyPayment-expenses${i}" name="modal-setMonthlyPayment-expenses${i}" placeholder="금액" value="${expenses}">
                    </div>
                    <div class="col ${pt}">
		                <input type="date" class="form-control" id="modal-setMonthlyPayment-date${i}" name="modal-setMonthlyPayment-date${i}" value="${data[i].Date}" max="9999-12-31">
		            </div>
                    <div class="col ${pt}">
		                <input type="date" class="form-control" id="modal-setMonthlyPayment-depositdate${i}" name="modal-setMonthlyPayment-depositdate${i}" value="${data[i].DepositDate}" max="9999-12-31">
		            </div>
                    <div class="col ${pt} text-right">
                        <div class="custom-control custom-radio custom-control-inline">
                            <input type="radio" id="modal-setMonthlyPayment-statusone${i}" name="modal-setMonthlyPayment-status${i}" class="custom-control-input" value="true" ${status1}>
                            <label class="custom-control-label text-muted" for="modal-setMonthlyPayment-statusone${i}">Yes</label>
                        </div>
                        <div class="custom-control custom-radio custom-control-inline">
                            <input type="radio" id="modal-setMonthlyPayment-statustwo${i}" name="modal-setMonthlyPayment-status${i}" class="custom-control-input" value="false" ${status2}>
                            <label class="custom-control-label text-muted" for="modal-setMonthlyPayment-statustwo${i}">No</label>
                        </div>
                    </div>
                    <div class="col-1 ${pt}">
                        <span class="badge badge-pill badge-danger text-center finger" onclick="delMonthlyPaymentFunc(${i})">Del</span>
                    </div>
                    `
                    e.innerHTML = html;
                    parent.appendChild(e);
                }
            }
            document.getElementById("modal-setMonthlyPayment-paymentNum").value = parent.childElementCount; // 최종 생성된 구매 내역 개수를 paymentNum에 저장한다.
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}`);
        }
    })
}

// addMonthlyPaymentFunc 함수는 + 버튼을 클릭하면 매출 내역을 입력할 수 있는 칸을 하나 더 늘려주는 함수이다.
function addMonthlyPaymentFunc() {
    let childNum = document.getElementById("modal-setMonthlyPayment-payment").childElementCount;
    let e = document.createElement("div");
    e.className = "form-row align-items-center";
    html = `
    <div class="col pt-2">
        <select id="modal-setMonthlyPayment-type${childNum}" name="modal-setMonthlyPayment-type${childNum}" class="form-control">
            <option value=""></option>
            <option value="계약금">계약금</option>
            <option value="중도금">중도금</option>
            <option value="잔금">잔금</option>
        </select>
	</div>
	<div class="col pt-2">
	    <input type="text" inputmode="numeric" class="form-control" id="modal-setMonthlyPayment-expenses${childNum}" name="modal-setMonthlyPayment-expenses${childNum}" placeholder="금액">
    </div>
	<div class="col pt-2">
	    <input type="date" class="form-control" id="modal-setMonthlyPayment-date${childNum}" name="modal-setMonthlyPayment-date${childNum}" max="9999-12-31">
	</div>
    <div class="col pt-2">
	    <input type="date" class="form-control" id="modal-setMonthlyPayment-depositdate${childNum}" name="modal-setMonthlyPayment-depositdate${childNum}" max="9999-12-31">
	</div>
    <div class="col pt-2 text-right">
        <div class="custom-control custom-radio custom-control-inline">
            <input type="radio" id="modal-setMonthlyPayment-statusone${childNum}" name="modal-setMonthlyPayment-status${childNum}" class="custom-control-input" value="true">
            <label class="custom-control-label text-muted" for="modal-setMonthlyPayment-statusone${childNum}">Yes</label>
        </div>
        <div class="custom-control custom-radio custom-control-inline">
            <input type="radio" id="modal-setMonthlyPayment-statustwo${childNum}" name="modal-setMonthlyPayment-status${childNum}" class="custom-control-input" value="false" checked>
            <label class="custom-control-label text-muted" for="modal-setMonthlyPayment-statustwo${childNum}">No</label>
        </div>
    </div>
    <div class="col-1 pt-2">
        <span class="badge badge-pill badge-danger text-center finger" onclick="delMonthlyPaymentFunc(${childNum})">Del</span>
    </div>
    `
    e.innerHTML = html;
    document.getElementById("modal-setMonthlyPayment-payment").appendChild(e);
    document.getElementById("modal-setMonthlyPayment-paymentNum").value = document.getElementById("modal-setMonthlyPayment-payment").childElementCount; // 최종 생성된 매출 내역 개수를 paymentNum에 저장한다.
}

// delMonthlyPaymentFunc 함수는 modal-setMonthlyPayment에서 Del 버튼을 클릭했을 때 입력창의 값들을 모두 초기화하는 함수이다.
function delMonthlyPaymentFunc(index) {
    document.getElementById(`modal-setMonthlyPayment-type${index}`).value = "";
    document.getElementById(`modal-setMonthlyPayment-expenses${index}`).value = "";
    document.getElementById(`modal-setMonthlyPayment-date${index}`).value = "";
    document.getElementById(`modal-setMonthlyPayment-depositdate${index}`).value = "";
    document.getElementById(`modal-setMonthlyPayment-statusone${index}`).checked = false;
    document.getElementById(`modal-setMonthlyPayment-statustwo${index}`).checked = false;
}

// setMonthlyPaymentFunc 함수는 modal-setMonthlyPayment에서 Update 버튼을 클릭하면 restAPI를 이용하여 월별 매출 내역을 업데이트하는 함수이다.
function setMonthlyPaymentFunc() {
    let token = document.getElementById("token").value;
    let id = document.getElementById("modal-setMonthlyPayment-projectID").value;
    let date = document.getElementById("modal-setMonthlyPayment-date").value;
    let num = document.getElementById("modal-setMonthlyPayment-paymentNum").value;

    var url = []
    for (var i = 0; i < num; i++) {
        var type = document.getElementById(`modal-setMonthlyPayment-type${i}`).value;
        var paymentDate = document.getElementById(`modal-setMonthlyPayment-date${i}`).value;
        var expenses = document.getElementById(`modal-setMonthlyPayment-expenses${i}`).value;
        var status1 = document.getElementById(`modal-setMonthlyPayment-statusone${i}`).checked;
        var status2 = document.getElementById(`modal-setMonthlyPayment-statustwo${i}`).checked;
        var depositDate = document.getElementById(`modal-setMonthlyPayment-depositdate${i}`).value;
        if (expenses.includes(",")) {
            expenses = expenses.replace(/,/gi, '');
        }
        if (type == "" && paymentDate == "" && expenses == "") {  // 타입, 날짜, 금액 모두 빈칸이면 continue
            continue
        }
        if (type == "") {
            alert("계약금, 중도금, 잔금 중 하나를 선택해주세요");
            return
        }
        if (paymentDate == "") {
            alert("세금계산서 발행일을 입력해주세요");
            return
        }
        // 세금계산서 발행일이 그 달에 포함되는지 확인한다.
        if (!paymentDate.includes(date)) {
            alert("세금계산서 발행일을 확인해주세요");
            return
        }
        if (expenses == "") {
            alert("금액을 입력해주세요");
            return
        }
        if (status1 == false && status2 == false) {
            alert("입금 여부를 선택해주세요");
            return
        }
        if (status1 == true && depositDate == "") {
            alert("입금일을 입력해주세요");
            return
        }
        if (depositDate != "") {
            depositDateType = new Date(depositDate);
            today = new Date();
            if (depositDateType <= today && status1 == false) {
                alert("입금 여부를 확인해주세요");
                return
            }
        }
        url.push(`type${i}=${type}&expenses${i}=${expenses}&date${i}=${paymentDate}&depositdate${i}=${depositDate}&status${i}=${status1}`)
    }
    url.push(`num=${num}`)

    // 월별 매출의 합이 총 매출보다 크지 않은지 확인한다.
    let totalPaymentNum = document.getElementById("paymentNum").value;
    let totalPayment = 0;
    for (var i = 0; i < totalPaymentNum; i++) {
        var payment = document.getElementById(`payment${i}`).value;
        if (payment.includes(",")) {
            payment = payment.replace(/,/gi, "");
        }
        totalPayment += Number(payment);
    }

    var startDate = new Date(document.getElementById("startdate").value);
    var endDate = new Date(document.getElementById("enddate").value);
    var curDate = new Date(document.getElementById("modal-setMonthlyPayment-date").value);
    var sumMonthlyPayment = 0;
    while(true) {
        var year = startDate.getFullYear().toString();
        var tempDate = new Date(startDate.getUTCFullYear(), startDate.getMonth() + 1) // 자바스크립트에서는 월이 0부터 시작한다. 
        var month = tempDate.getMonth().toString()
        if (tempDate.getMonth() == 0) {
            month = "12"
        }
        var sd = year + "-" + month.padStart(2, "0") + "monthlypayment";
        if (document.getElementById(sd) != null) { // 작업기간을 수정한 경우 존재하지 않는 칸의 value를 가져올 때 콘솔 로그에 에러메시지가 나온다.
            // 현재 월별 매출을 입력하고 있는 달이면 모달창에 있는 매출의 합을 구한다.
            if (curDate.getFullYear() == startDate.getFullYear() && curDate.getMonth() == startDate.getMonth()) {
                var monthlyPaymentNum = document.getElementById("modal-setMonthlyPayment-paymentNum").value;
                for (var i = 0; i < monthlyPaymentNum; i++) {
                    var monthlyPayment = document.getElementById(`modal-setMonthlyPayment-expenses${i}`).value;
                    if (monthlyPayment.includes(",")) {
                        monthlyPayment = monthlyPayment.replace(/,/gi, "");
                    }
                    sumMonthlyPayment = sumMonthlyPayment + Number(monthlyPayment);
                }
            } else { // 현재 월별 매출을 입력하고 있는 달이 아니면 페이지에 있는 월별 매출의 합을 구한다.
                let monthlyPayment = document.getElementById(sd).value;
                if (monthlyPayment.includes(",")) {
                    monthlyPayment = monthlyPayment.replace(/,/gi, '')
                }
                sumMonthlyPayment = sumMonthlyPayment + Number(monthlyPayment)
            }
        }
        if (startDate.getFullYear()== endDate.getFullYear() && startDate.getMonth() == endDate.getMonth()) {
            break
        }
        startDate.setMonth(startDate.getMonth() + 1)
    }

    if (totalPayment < sumMonthlyPayment) {
        alert("월별 매출의 합이 총 매출보다 큽니다.\n다시 확인해주세요.");
        return
    }

    $.ajax({
        url: `/api/setMonthlyPayment?id=${id}&date=${date}&${url.join("&")}`,
        type: "post",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        success: function(data) {
            alert("매출 내역이 저장되었습니다.");
            $("#modal-setMonthlyPayment").modal("hide");
            if (data == "0") {  // 총합이 0이면 빈 문자열을 보여준다.
                data = "";
            }
            let expenses = data.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",")
            document.getElementById(`${date}monthlypayment`).value = expenses;  // 총액
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}`);
        }
    })
}

// updateProjectsFunc 함수는 샷건에서 프로젝트를 업데이트하는 함수이다.
function updateProjectsFunc() {
    let token = document.getElementById("token").value;
    $("#modal-updateprojects").modal("show");

    $.ajax({
        url: `/api/updateprojects`,
        type: "post",
        headers: {
            "Authorization": "Basic " + token,
        },
        success: function(){
            $("#modal-updateprojects").modal("hide");
            window.location.reload()
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}\n\n프로젝트를 업데이트하는 동안 문제가 발생하였습니다`)
        }
    })
}

/* 메인 페이지 관련 함수 */
// setRndCheckBoxFunc 함수는 기타 프로젝트 체크박스를 체크하면 다른 체크박스를 해제하는 함수이다.
function setRndCheckBoxFunc(changedCheckBoxID, checkBoxID) {
    var isChecked = document.getElementById(changedCheckBoxID).checked;
    if (isChecked == false) {
        return
    }
    document.getElementById(checkBoxID).checked = false;
}

// setInitUrlFunc 함수는 init 페이지에서 sort를 눌렀을 때 url을 바꿔주는 함수이다.
// 나중에 사용될 수도 있는 함수여서 남겨두기로 했다. 지금 사용되는 곳이 없으니 헷갈릴 수 있어 주석처리했다.
/* function setInitUrlFunc(sort) {
    var url = document.URL;
    if (!url.includes("?")) {  // url에 파라미터가 없을 때
        url = url + "?sort=" + sort;
    } else { // url에 파라미터가 있을 때
        if (url.includes("sort")) {  // sort 키워드가 있으면 value만 바꿔준다.
            var regExp = new RegExp("sort=[a-z]+", "i");
            url = url.replace(regExp, "sort="+sort);
        } else {  // sort 키워드가 없으면 키워드를 추가한다.
            url = url + "&sort=" + sort;
        }
    }
    window.location.href = url;
} */

// setPopupNotDisplayFunc 함수는 팝업창을 하루동안 보이지 않게 쿠키에 설정하는 함수이다.
function setPopupNotDisplayFunc(id) {
    document.getElementById(id).style.display = "none"; // 팝업창을 닫는다.
    let date = new Date();
    date.setHours(24, 0, 0, 0); // 다음날 자정
    document.cookie = `${id}=no; expires=${date}`  // 쿠키에 저장
}

/* AdminSetting 관련 함수 */
// addProjectStatusFunc 함수는 프로젝트 Status를 입력하는 창을 추가하는 함수이다.
function addProjectStatusFunc() {
    let childNum = document.getElementById("projectStatus").childElementCount;
    let e = document.createElement("div");
    let html = `
    <div class="row pt-2">
        <div class="col">
            <label class="text-muted">Status Name</label>
            <input type="text" name="statusid${childNum}" class="form-control">
        </div>
        <div class="col">
            <label class="text-muted">Text</label>
            <input class="jscolor{valueElement:'textcolor${childNum}'} btn border border-light w-100">
        </div>
        <div class="col">
            <label class="text-muted">BG</label>
            <input class="jscolor{valueElement:'bgcolor${childNum}'} btn border border-light w-100">
        </div>
    </div>
    <div class="row pt-2">
        <div class="col"></div>
        <div class="col">
            <input id="textcolor${childNum}" name="textcolor${childNum}" type="text" class="form-control" value="ffffff">
            <small class="form-text text-muted">Status 글자색</small>
        </div>
        <div class="col">
            <input id="bgcolor${childNum}" name="bgcolor${childNum}" type="text" class="form-control" value="6b605d">
            <small class="form-text text-muted">Status 배경색</small>
        </div>
    </div>
    `
    e.innerHTML = html;
    document.getElementById("projectStatus").appendChild(e);
    document.getElementById("projectStatusNum").value = document.getElementById("projectStatus").childElementCount;
}

/* Vendor 관련 함수 */
// checkAddVendorPageFunc 함수는 Add Vendor 페이지에서 빈칸을 체크하는 함수이다.
function checkAddVendorPageFunc() {
    // 필수 정보 체크
    if (document.getElementById("project").value == "") {
        alert("프로젝트를 선택해주세요")
        return false
    }
    if (document.getElementById("name").value == "") {
        alert("벤더명을 적어주세요")
        return false
    }
    if (document.getElementById("expenses").value == "") {
        alert("총 비용을 입력해주세요")
        return false
    }
    if (document.getElementById("date").value == "") {
        alert("계약일을 입력해주세요")
        return false
    }

    // 부가 정보 체크
    let blank_pattern = /[\s]/g;
    if (document.getElementById("tasks").value != "") {
        if(blank_pattern.test(document.getElementById("tasks").value) == true) {
            alert("태스크에 공백은 허용하지 않습니다")
            return false
        }
    }

    // 금액 관련 체크 - 금액을 계약금 및 잔금 중 하나라도 적었는지 확인
    let downpayment = document.getElementById("downpayment").value
    let balance = document.getElementById("balance").value

    if (downpayment == "" && balance == "") {
        alert("계약금 혹은 잔금 중에 하나는 필수로 입력해주셔야 합니다")
        return false
    }
    
    // 금액을 적었다면 세금 계산서 발행 날짜를 입력했는지 확인
    if (downpayment != "" && document.getElementById("downpaymentdate").value == "") {
        alert("계약금이 있는 경우 계약금 세금 계산서 발행일을 입력해주세요")
        return false
    }
    let mediumplatingNum = Number(document.getElementById("mediumplatingNum").value);
    for (let num = 0; num < mediumplatingNum; num++) {
        let mediumplatingID = "mediumplating" + String(num)
        let mediumplatingdateID = "mediumplatingdate" + String(num)
        let mediumplating = document.getElementById(mediumplatingID).value
        let mediumplatingdate = document.getElementById(mediumplatingdateID).value
        if (mediumplating != "" && mediumplatingdate == "" ) {
            alert("중도금이 있는 경우 중도금 세금 계산서 발행일을 입력해주세요")
            return false
        }
    }
    if (balance != "" && document.getElementById("balancedate").value == "") {
        alert("잔금이 있는 경우 잔금 세금 계산서 발행일을 입력해주세요")
        return false
    }

    // 계약금 지급일과 지급여부 확인
    let now = new Date();
    let dpPayedDate = new Date(document.getElementById("downpaymentpayeddate").value);
    let dpStatus = document.getElementById("downpaymentstatus1");
    if (document.getElementById("downpaymentpayeddate").value != "" && !(dpPayedDate > now) && dpStatus.checked == false) { // 지급일이 적혀있는데 지급일이 오늘보다 전이면 지급 여부가 Yes인지 확인
        alert("계약금 지급일이 지났습니다. 지급 여부를 확인해주세요");
        return false;
    }
    if (dpStatus.checked == true){ // 지급 여부가 Yes인데 지급일이 적혀있는지 적혀있다면 오늘보다 전인지 확인
        if (document.getElementById("downpaymentpayeddate").value == "") {
            alert("계약금 지급 여부가 Yes입니다. 계약금 지급일을 입력해주세요");
            return false;
        } else if (dpPayedDate > now) {
            alert("계약금 지급일이 오늘 이후로 입력되었습니다. 계약금 지급일을 확인해주세요");
            return false;
        }
    }

    // 중도금 지급일과 지급여부 확인
    for (let num = 0; num < mediumplatingNum; num++) {
        let mpPayedDateID = "mediumplatingpayeddate" + String(num);
        let mpStatusID = "mediumplatingstatus1" + String(num);
        let mpPayedDate = new Date(document.getElementById(mpPayedDateID).value);
        let mpStatus = document.getElementById(mpStatusID);
        if (document.getElementById(mpPayedDateID).value != "" && !(mpPayedDate > now) && mpStatus.checked == false) { // 지급일이 적혀있는데 지급일이 오늘보다 전이면 지급 여부가 Yes인지 확인
            alert("중도금 지급일이 지났습니다. 지급 여부를 확인해주세요");
            return false;
        }
        if (mpStatus.checked == true){ // 지급 여부가 Yes인데 지급일이 적혀있는지 적혀있다면 오늘보다 전인지 확인
            if (document.getElementById(mpPayedDateID).value == "") {
                alert("중도금 지급 여부가 Yes입니다. 중도금 지급일을 입력해주세요");
                return false;
            } else if (mpPayedDate > now) {
                alert("중도금 지급일이 오늘 이후로 입력되었습니다. 중도금 지급일을 확인해주세요");
                return false;
            }
        }
    }

    // 잔금 지급일과 지급여부 확인
    let blPayedDate = new Date(document.getElementById("balancepayeddate").value);
    let blStatus = document.getElementById("balancestatus1");
    if (document.getElementById("balancepayeddate").value != "" && !(blPayedDate > now) && blStatus.checked == false) { // 지급일이 적혀있는데 지급일이 오늘보다 전이면 지급 여부가 Yes인지 확인
        alert("잔금 지급일이 지났습니다. 지급 여부를 확인해주세요");
        return false;
    }
    if (blStatus.checked == true){ // 지급 여부가 Yes인데 지급일이 적혀있는지 적혀있다면 오늘보다 전인지 확인
        if (document.getElementById("balancepayeddate").value == "") {
            alert("잔금 지급 여부가 Yes입니다. 잔금 지급일을 입력해주세요");
            return false;
        } else if (blPayedDate > now) {
            alert("잔금 지급일이 오늘 이후로 입력되었습니다. 잔금 지급일을 확인해주세요");
            return false;
        }
    }

    // 적은 총 금액의 합이 총 비용과 같은지 확인
    let expensesSum = 0
    if (downpayment.includes(",")) {
        downpayment = downpayment.replace(/,/gi, '')
    }
    expensesSum = expensesSum + Number(downpayment)
    for (let num = 0; num < mediumplatingNum; num++) {
        let mediumplatingID = "mediumplating" + String(num)
        let mediumplating = document.getElementById(mediumplatingID).value
        if (mediumplating.includes(",")) {
            mediumplating = mediumplating.replace(/,/gi, '')
        }
        expensesSum = expensesSum + Number(mediumplating)
    } 
    if (balance.includes(",")) {
        balance = balance.replace(/,/gi, '')
    }
    expensesSum = expensesSum + Number(balance)
    let expenses = document.getElementById("expenses").value
    if (expenses.includes(",")) {
        expenses = expenses.replace(/,/gi, '')
    }
    let expensesTotal = Number(expenses)
    if (expensesTotal != expensesSum) {
        alert("총 비용과 입력한 금액의 합이 다릅니다.")
        return false
    }
}

// addMediumPlatingFunc 함수는 Add Vendor 페이지에서 중도금 입력칸을 추가하는 함수이다.
function addMediumPlatingFunc() {
    let childNum = document.getElementById("addmediumplating").childElementCount;
    let e = document.createElement("div");
    let html = `
    <div class="row pt-2">
        <div class="col">
            <div class="form-group pb-2">
                <label class="text-muted">중도금${childNum + 1}</label>
                <input type="text" inputmode="numeric" class="form-control" id="mediumplating${childNum}" name="mediumplating${childNum}">
            </div>
        </div>
        <div class="col">
            <div class="form-group pb-2">
                <label class="text-muted">세금 계산서 발행일</label>
                <input type="date" class="form-control" id="mediumplatingdate${childNum}" name="mediumplatingdate${childNum}" max="9999-12-31">
            </div>
        </div>
        <div class="col">
            <div class="form-group pb-2">
                <label class="text-muted">지급일</label>
                <input type="date" class="form-control" id="mediumplatingpayeddate${childNum}" name="mediumplatingpayeddate${childNum}" max="9999-12-31">
            </div>
        </div>
        <div class="col">
            <div class="row mt-4 pt-2 pb-2">
                <div class="col-5">
                    <label class="text-muted">지급 여부</label>
                </div>
                <div class="col">
                    <div class="form-group">
                        <div class="custom-control custom-radio custom-control-inline">
                            <input type="radio" id="mediumplatingstatus1${childNum}" name="mediumplatingstatus${childNum}" class="custom-control-input" value="true">
                            <label class="custom-control-label text-muted" for="mediumplatingstatus1${childNum}">Yes</label>
                        </div>
                        <div class="custom-control custom-radio custom-control-inline">
                            <input type="radio" id="mediumplatingstatus2${childNum}" name="mediumplatingstatus${childNum}" class="custom-control-input" value="false" checked>
                            <label class="custom-control-label text-muted" for="mediumplatingstatus2${childNum}">No</label>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
    `
    e.innerHTML = html;
    document.getElementById("addmediumplating").appendChild(e);
    document.getElementById("mediumplatingNum").value = document.getElementById("addmediumplating").childElementCount;
}

// setRmVendorModalFunc 함수는 Vendor 삭제 모달에 값을 넣는 함수이다.
function setRmVendorModalFunc(id, project, name) {
    document.getElementById("modal-rmvendor-id").value = id
    document.getElementById("modal-rmvendor-project").value = project
    document.getElementById("modal-rmvendor-name").value = name
}

// rmVendorFunc 함수는 Vendor를 삭제하는 함수이다.
function rmVendorFunc(id, project, name) {
    let token = document.getElementById("token").value;
    $.ajax({
        url: `/api/rmvendor?id=${id}&project=${project}&name=${name}`,
        type: "delete",
        headers: {
            "Authorization": "Basic " + token,
        },
        success: function() {
            alert("Vendor가 삭제되었습니다.")
            location.reload();  // 페이지 새로고침
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}`);
        }
    })
}

/* 예산 TeamSetting 관련 함수 */
// addTeamSettingTaskFunc 함수는 예산 TeamSetting에서 부서에 해당하는 구분과 태스크를 추가하는 함수이다.
function addTeamSettingTaskFunc(head, dept) {
    let id = `head${head}-dept${dept}-part`;
    let childNum = document.getElementById(id).childElementCount;
    let e = document.createElement("div");
    let html = `
    <div class="row">
        <div class="col-4">
            <div class="form-group">
                <label class="text-muted" for="head${head}-dept${dept}-part${childNum}">구분</label>
                <input type="text" name="head${head}-dept${dept}-part${childNum}" id="head${head}-dept${dept}-part${childNum}" class="form-control">
                <small class="form-text text-muted">예산에서 해당 부서에 속하는 구분을 입력해주세요.</small>
            </div>
        </div>
        <div class="col-8">
            <div class="form-group">
                <label class="text-muted" for="head${head}-dept${dept}-task${childNum}">태스크</label>
                <input type="text" name="head${head}-dept${dept}-task${childNum}" id="head${head}-dept${dept}-task${childNum}" class="form-control">
                <small class="form-text text-muted">예산에서 해당 구분에 속하는 태스크들을 입력해주세요. 띄어쓰기로 구분합니다.</small>
            </div>
        </div>
    </div>
    `
    e.innerHTML = html;
    document.getElementById(id).appendChild(e);
    document.getElementById(id + "num").value = document.getElementById(id).childElementCount;
}

// addTeamSettingControlTeamFunc 함수는 예산 TeamSetting에서 부서에 해당하는 관리 구분과 관리 팀 콤보박스를 추가하는 함수이다.
function addTeamSettingControlTeamFunc(head, control, teams) {
    let id = `head${head}-control${control}-team`;
    let childNum = document.getElementById(id).childElementCount;
    let e = document.createElement("div");

    // javascript에서 추가된 팀 버튼의 경우 팀 리스트가 string으로 들어오기 때문에 ","으로 split해준다.
    if (typeof(teams) == "string") {
        teams = teams.split(",")
    }

    // 전달받은 팀 목록을 select에 넣어줄 옵션 형식으로 바꿔준다.
    let selectHtml = "";
    for (var i = 0; i < teams.length; i++) {
        selectHtml += `<option value="${teams[i]}">${teams[i]}</option>\n`
    }

    let html = `
    <div class="row">
        <div class="col-4">
            <div class="form-group">
                <label class="text-muted" for="head${head}-control${control}-part${childNum}">구분</label>
                <input type="text" name="head${head}-control${control}-part${childNum}" id="head${head}-control${control}-part${childNum}" class="form-control">
                <small class="form-text text-muted">예산에서 해당 부서에 속하는 구분을 입력해주세요.</small>
            </div>
        </div>
        <div class="col-8">
            <div class="form-group">
                <label class="text-muted" for="head${head}-control${control}-team${childNum}">팀</label>
                <select name="head${head}-control${control}-team${childNum}" id="head${head}-control${control}-team${childNum}" class="form-control teamselect" multiple="multiple">
                    ${selectHtml}
                </select>  
                <small class="form-text text-muted">예산에서 해당 구분에 속하는 팀들을 선택해주세요.</small>
            </div>
        </div>
    </div>
    `
    e.innerHTML = html;
    document.getElementById(id).appendChild(e);
    document.getElementById(id + "num").value = document.getElementById(id).childElementCount;

    $(`.teamselect`).select2();
}

// addTeamSettingPartFunc 함수는 예산 TeamSetting에서 본부에 해당하는 부서를 추가하는 함수이다.
function addTeamSettingPartFunc(head) {
    let id = `head${head}-dept`;
    let childNum = document.getElementById(id).childElementCount;
    let e = document.createElement("div");
    let html = `
    <div class="row">
        <div class="col-2">
            <div class="form-group custom-control custom-checkbox custom-control-lg" style="margin-left: 80px; margin-top: 35px;">
                <input type="checkbox" class="custom-control-input" id="head${head}-type${childNum}" name="head${head}-type${childNum}">
                <label class="custom-control-label text-muted" for="head${head}-type${childNum}">Asset</label>
            </div>
        </div>
        <div class="col-3">
            <div class="form-group">
                <label class="text-muted" for="head${head}-dept${childNum}">부서</label>
                <input type="text" name="head${head}-dept${childNum}" id="head${head}-dept${childNum}" class="form-control">
                <small class="form-text text-muted">예산에서 해당 본부에 속하는 부서를 입력해주세요.</small>
            </div>
        </div>
        <div class="col-7">
            <div id="head${head}-dept${childNum}-part">
            </div>
            <div class="row">
                <div class="col">
                    <!-- 태스크 추가 버튼 -->
                    <input type="hidden" id="head${head}-dept${childNum}-partnum" name="head${head}-dept${childNum}-partnum" value="1">
                    <span id="taskaddbtn" class="add float-right mt-2" onclick="addTeamSettingTaskFunc(${head}, ${childNum});">태스크</span>
                </div>
            </div>
        </div>
    </div>
    `
    e.innerHTML = html;
    document.getElementById(id).appendChild(e);
    document.getElementById(id + "num").value = document.getElementById(id).childElementCount;

    addTeamSettingTaskFunc(head, childNum); // 태스크 레이아웃을 생성한다.
}

// addTeamSettingControlFunc 함수는 예산 TeamSetting에서 본부에 해당하는 관리 부서를 추가하는 함수이다.
function addTeamSettingControlFunc(head, teams) {
    let id = `head${head}-control`;
    let childNum = document.getElementById(id).childElementCount;
    let e = document.createElement("div");

    let html=`
    <div class="row">
        <div class="col-2"></div>
        <div class="col-3">
            <div class="form-group">
                <label class="text-muted" for="head${head}-control${childNum}">부서(SUP)</label>
                <input type="text" name="head${head}-control${childNum}" id="head${head}-control${childNum}" class="form-control">
                <small class="form-text text-muted">예산에서 해당 본부에 속하는 부서를 입력해주세요.</small>
            </div>
        </div>
        <div class="col-7">
            <div id="head${head}-control${childNum}-team">
            </div>
            <div class="row">
                <div class="col">
                    <!-- 팀 추가 버튼 -->
                    <input type="hidden" id="head${head}-control${childNum}-teamnum" name="head${head}-control${childNum}-teamnum" value="1">
                    <span id="teamaddbtn" class="add float-right mt-2" onclick="addTeamSettingControlTeamFunc(${head}, ${childNum}, '${teams}');">팀</span>
                </div>
            </div>
        </div>
    </div>
    `
    e.innerHTML = html;
    document.getElementById(id).appendChild(e);
    document.getElementById(id + "num").value = document.getElementById(id).childElementCount;

    addTeamSettingControlTeamFunc(head, childNum, teams);
}

// addTeamSettingHeadFunc 함수는 예산 TeamSetting에서 본부를 추가하는 함수이다.
function addTeamSettingHeadFunc(teams) {
    let childNum = document.getElementById("teamSetting").childElementCount;
    let e = document.createElement("div");
    let html = `
    <div class="row pt-3">
        <div class="col-2">
            <div class="form-group">
                <label class="text-muted" for="head${childNum}">본부</label>
                <input type="text" name="head${childNum}" id="head${childNum}" class="form-control">
                <small class="form-text text-muted">예산에서 사용될 본부를 입력해주세요.</small>
            </div>
        </div>
        <div class="col-10">
            <div id="head${childNum}-dept"> 
            </div>
            <div id="head${childNum}-control">
            </div>
            <div class="row">
                <div class="col-5">
                    <!-- Part / Sup 추가 버튼 -->
                    <input type="hidden" id="head${childNum}-deptnum" name="head${childNum}-deptnum" value="1">
                    <input type="hidden" id="head${childNum}-controlnum" name="head${childNum}-controlnum" value="0">
                    <span id="supaddbtn" class="add float-right mt-2" onclick="addTeamSettingControlFunc('${childNum}', '${teams}');">SUP</span>
                    <span id="partaddbtn" class="add float-right mr-2 mt-2" onclick="addTeamSettingPartFunc(${childNum});">부서</span>
                </div>
                <div class="col-7"></div>
            </div>
        </div>
    </div>
    `
    e.innerHTML = html;
    document.getElementById("teamSetting").appendChild(e);
    document.getElementById("headnum").value = document.getElementById("teamSetting").childElementCount;

    addTeamSettingPartFunc(childNum); // 부서, 태스크 레이아웃을 생성한다.
}

/* 예산 프로젝트 관련 함수 */
// setRmBGProjectModalFunc 함수는 예산 프로젝트 삭제 버튼을 클릭하면 ID, 이름을 받아 modal 창에 보여주는 함수이다.
function setRmBGProjectModalFunc(id, name) {
    document.getElementById("modal-rmbgproject-id").value = id;
    document.getElementById("modal-rmbgproject-name").value = name;
}

// rmBGProjectFunc 함수는 restAPI를 이용하여 예산 프로젝트를 삭제하는 함수이다.
function rmBGProjectFunc(id) {
    let token = document.getElementById("token").value;

    $.ajax({
        url: `/api/rmbgproject?id=${id}`,
        type: "delete",
        headers: {
            "Authorization": "Basic " + token,
        },
        dataType: "json",
        success: function(data) {
            alert("프로젝트가 삭제되었습니다.")
            location.reload();  // 페이지 새로고침
        },
        error: function(request, status, error) {
            alert(`code: ${request.status}\nstatus: ${status}\nmsg: ${request.responseText}\nerror: ${error}`);
        }
    })
}

// calNegoRatioFunc 함수는 입력 받은 제안 견적과 계약 결정액을 통해서 네고율을 계산하는 함수이다.
function calNegoRatioFunc(id) {
    let proposalid = "bgproposal"
    let decisionid = "bgdecision"
    let negoid = "bgnegoratio"
    if (id != '') {
        proposalid = id + "-" + proposalid
        decisionid = id + "-" + decisionid
        negoid = id + "-" + negoid
    }

    let bgproposal = document.getElementById(proposalid).value
    let bgdecision = document.getElementById(decisionid).value
    if (bgproposal == "" || bgdecision == "") {
        document.getElementById(negoid).value = ""
    }
    else {
        if (bgproposal.includes(",")) {
            bgproposal = bgproposal.replace(/,/gi, '')
        }
    
        if (bgdecision.includes(",")) {
            bgdecision = bgdecision.replace(/,/gi, '')
        }
    
        let negoRatio = Math.round((Number(bgproposal) - Number(bgdecision)) / Number(bgproposal) * 100);
        document.getElementById(negoid).value = negoRatio
    }
}

// addBGSupervisorSettingFunc 함수는 예산 프로젝트 추가 및 수정 페이지에서 수퍼바이저 정보를 입력칸을 추가하는 함수이다.
function addBGSupervisorSettingFunc(supervisor, id) {
    if(id != '') {
        id = id + "-"
    }
    let childNum = document.getElementById(id + "bgsupervisor").childElementCount;
    let e = document.createElement("div");

    if (typeof(supervisor) == "string") {
        let supList = supervisor.split("/")

        let supInfoList = []
        for (var i = 0; i < supList.length; i++) {
            let supInfo = supList[i].split(",");
            var sup = {}
            for (var j = 0; j < supInfo.length; j++) {
                sup[supInfo[j].split(":")[0]] = supInfo[j].split(":")[1]
            }
            supInfoList.push(sup)
        }
        supervisor = supInfoList
    }

    // 전달받은 아티스트 목록을 select에 넣어줄 옵션 형식으로 바꿔준다.
    let selectHtml = "";
    for (var i = 0; i < supervisor.length; i++) {
        selectHtml += `<option value="${supervisor[i].ID}">${supervisor[i].Name}</option>\n`
    }

    let html = ``
    if (childNum == 0) {
        html = 
        `
        <div class="row pt-3 pb-2">
            <div class="col form-group">
                <label class="text-muted">수퍼바이저</label>
                <select name="${id}supervisor${childNum}" id="${id}supervisor${childNum}" class="form-control custom-select left-radius right-radius">
                    <option value=""></option>
                    ${selectHtml}
                </select>
            </div>
            <div class="col form-group">
                <label class="text-muted">업무</label>
                <input type="text" class="form-control" id="${id}supervisor${childNum}-bgmanagementwork" name="${id}supervisor${childNum}-bgmanagementwork">
            </div>
            <div class="col form-group">
                <label class="text-muted">참여 기간</label>
                <input type="number" class="form-control" id="${id}supervisor${childNum}-bgmanagementperiod" name="${id}supervisor${childNum}-bgmanagementperiod">
                <small id="${id}supervisor${childNum}-bgmanagementperiod-label" class="form-text text-muted">숫자만 입력해주세요.</small>
            </div>
            <div class="col form-group">
                <label class="text-muted">참여 기간 퍼센티지</label>
                <div class="input-group">
                    <input type="number" class="form-control" id="${id}supervisor${childNum}-bgmanagementratio" name="${id}supervisor${childNum}-bgmanagementratio" step="0.01" max="100">
                    <div class="input-group-append">
                        <span class="input-group-text" style="background-color: #1C1C1C!important; color: #A7A59C!important; border: 1px solid #A7A59C!important;">%</span>
                    </div>
                </div>
            </div>
        </div>
        `
    } else {
        html = 
        `
        <div class="row pb-2">
            <div class="col form-group">
                <select name="${id}supervisor${childNum}" id="${id}supervisor${childNum}" class="form-control custom-select left-radius right-radius">
                    <option value=""></option>
                    ${selectHtml}
                </select>
            </div>
            <div class="col form-group">
                <input type="text" class="form-control" id="${id}supervisor${childNum}-bgmanagementwork" name="${id}supervisor${childNum}-bgmanagementwork">
            </div>
            <div class="col form-group">
                <input type="number" class="form-control" id="${id}supervisor${childNum}-bgmanagementperiod" name="${id}supervisor${childNum}-bgmanagementperiod">
                <small id="${id}supervisor${childNum}-bgmanagementperiod-label" class="form-text text-muted">숫자만 입력해주세요.</small>
            </div>
            <div class="col form-group">
                <div class="input-group">
                    <input type="number" class="form-control" id="${id}supervisor${childNum}-bgmanagementratio" name="${id}supervisor${childNum}-bgmanagementratio" step="0.01" max="100">
                    <div class="input-group-append">
                        <span class="input-group-text" style="background-color: #1C1C1C!important; color: #A7A59C!important; border: 1px solid #A7A59C!important;">%</span>
                    </div>
                </div>
            </div>
        </div>
        `
    }

    e.innerHTML = html;
    document.getElementById(id + "bgsupervisor").appendChild(e);
    document.getElementById(id + "bgsupervisornum").value = document.getElementById(id + "bgsupervisor").childElementCount;

    // 마지막 참여 기간을 제외한 다른 참여 기간들은 아래의 라벨을 지운다.
    document.getElementById(`${id}supervisor${childNum - 1}-bgmanagementperiod-label`).innerHTML = "";
}

// addBGProductionSettingFunc 함수는 예산 프로젝트 추가 및 수정 페이지에서 프로덕션 정보를 입력칸을 추가하는 함수이다.
function addBGProductionSettingFunc(production, id) {
    if(id != '') {
        id = id + "-"
    }
    let childNum = document.getElementById(id + "bgproduction").childElementCount;
    let e = document.createElement("div");

    if (typeof(production) == "string") {
        let prodList = production.split("/")

        let prodInfoList = []
        for (var i = 0; i < prodList.length; i++) {
            let prodInfo = prodList[i].split(",");
            var prod = {}
            for (var j = 0; j < prodInfo.length; j++) {
                prod[prodInfo[j].split(":")[0]] = prodInfo[j].split(":")[1]
            }
            prodInfoList.push(prod)
        }
        production = prodInfoList
    }
    
    // 전달받은 아티스트 목록을 select에 넣어줄 옵션 형식으로 바꿔준다.
    let selectHtml = "";
    for (var i = 0; i < production.length; i++) {
        selectHtml += `<option value="${production[i].ID}">${production[i].Name}</option>\n`
    }

    let html = ``
    if (childNum == 0) {
        html = 
        `
        <div class="row pt-5 pb-2">
            <div class="col form-group">
                <label class="text-muted">프로덕션</label>
                <select name="${id}production${childNum}" id="${id}production${childNum}" class="form-control custom-select left-radius right-radius">
                    <option value=""></option>
                    ${selectHtml}
                </select>
            </div>
            <div class="col form-group">
                <label class="text-muted">업무</label>
                <input type="text" class="form-control" id="${id}production${childNum}-bgmanagementwork" name="${id}production${childNum}-bgmanagementwork">
            </div>
            <div class="col form-group">
                <label class="text-muted">참여 기간</label>
                <input type="number" class="form-control" id="${id}production${childNum}-bgmanagementperiod" name="${id}production${childNum}-bgmanagementperiod">
                <small id="${id}production${childNum}-bgmanagementperiod-label" class="form-text text-muted">숫자만 입력해주세요.</small>
            </div>
            <div class="col form-group">
                <label class="text-muted">참여 기간 퍼센티지</label>
                <div class="input-group">
                    <input type="number" class="form-control" id="${id}production${childNum}-bgmanagementratio" name="${id}production${childNum}-bgmanagementratio" step="0.01" max="100">
                    <div class="input-group-append">
                        <span class="input-group-text" style="background-color: #1C1C1C!important; color: #A7A59C!important; border: 1px solid #A7A59C!important;">%</span>
                    </div>
                </div>
            </div>
        </div>
        `
    } else {
        html = 
        `
        <div class="row pb-2">
            <div class="col form-group">
                <select name="${id}production${childNum}" id="${id}production${childNum}" class="form-control custom-select left-radius right-radius">
                    <option value=""></option>
                    ${selectHtml}
                </select>
            </div>
            <div class="col form-group">
                <input type="text" class="form-control" id="${id}production${childNum}-bgmanagementwork" name="${id}production${childNum}-bgmanagementwork">
            </div>
            <div class="col form-group">
                <input type="number" class="form-control" id="${id}production${childNum}-bgmanagementperiod" name="${id}production${childNum}-bgmanagementperiod">
                <small id="${id}production${childNum}-bgmanagementperiod-label" class="form-text text-muted">숫자만 입력해주세요.</small>
            </div>
            <div class="col form-group">
                <div class="input-group">
                    <input type="number" class="form-control" id="${id}production${childNum}-bgmanagementratio" name="${id}production${childNum}-bgmanagementratio" step="0.01" max="100">
                    <div class="input-group-append">
                        <span class="input-group-text" style="background-color: #1C1C1C!important; color: #A7A59C!important; border: 1px solid #A7A59C!important;">%</span>
                    </div>
                </div>
            </div>
        </div>
        `
    }
    
    e.innerHTML = html;
    document.getElementById(id + "bgproduction").appendChild(e);
    document.getElementById(id + "bgproductionnum").value = document.getElementById(id + "bgproduction").childElementCount;

    // 마지막 참여 기간을 제외한 다른 참여 기간들은 아래의 라벨을 지운다.
    document.getElementById(`${id}production${childNum - 1}-bgmanagementperiod-label`).innerHTML = "";
}

// addBGManagementSettingFunc 함수는 예산 프로젝트 추가 페이지에서 매니지먼트 정보를 입력칸을 추가하는 함수이다.
function addBGManagementSettingFunc(management, id) {
    if(id != '') {
        id = id + "-"
    }
    let childNum = document.getElementById(id + "bgmanagement").childElementCount;
    let e = document.createElement("div");
    
    if (typeof(management) == "string") {
        let mngList = management.split("/")

        let mngInfoList = []
        for (var i = 0; i < mngList.length; i++) {
            let mngInfo = mngList[i].split(",");
            var mng = {}
            for (var j = 0; j < mngInfo.length; j++) {
                mng[mngInfo[j].split(":")[0]] = mngInfo[j].split(":")[1]
            }
            mngInfoList.push(mng)
        }
        management = mngInfoList
    }

    // 전달받은 아티스트 목록을 select에 넣어줄 옵션 형식으로 바꿔준다.
    let selectHtml = "";
    for (var i = 0; i < management.length; i++) {
        selectHtml += `<option value="${management[i].ID}">${management[i].Name}</option>\n`
    }

    let html = ``
    if (childNum == 0) {
        html = 
        `
        <div class="row pt-5 pb-2">
            <div class="col form-group">
                <label class="text-muted">매니지먼트</label>
                <select name="${id}management${childNum}" id="${id}management${childNum}" class="form-control custom-select left-radius right-radius">
                    <option value=""></option>
                    ${selectHtml}
                </select>
            </div>
            <div class="col form-group">
                <label class="text-muted">업무</label>
                <input type="text" class="form-control" id="${id}management${childNum}-bgmanagementwork" name="${id}management${childNum}-bgmanagementwork">
            </div>
            <div class="col form-group">
                <label class="text-muted">참여 기간</label>
                <input type="number" class="form-control" id="${id}management${childNum}-bgmanagementperiod" name="${id}management${childNum}-bgmanagementperiod">
                <small id="${id}management${childNum}-bgmanagementperiod-label" class="form-text text-muted">숫자만 입력해주세요.</small>
            </div>
            <div class="col form-group">
                <label class="text-muted">참여 기간 퍼센티지</label>
                <div class="input-group">
                    <input type="number" class="form-control" id="${id}management${childNum}-bgmanagementratio" name="${id}management${childNum}-bgmanagementratio" step="0.01" max="100">
                    <div class="input-group-append">
                        <span class="input-group-text" style="background-color: #1C1C1C!important; color: #A7A59C!important; border: 1px solid #A7A59C!important;">%</span>
                    </div>
                </div>
            </div>
        </div>
        `
    } else {
        html = 
        `
        <div class="row pb-2">
            <div class="col form-group">
                <select name="${id}management${childNum}" id="${id}management${childNum}" class="form-control custom-select left-radius right-radius">
                    <option value=""></option>
                    ${selectHtml}
                </select>
            </div>
            <div class="col form-group">
                <input type="text" class="form-control" id="${id}management${childNum}-bgmanagementwork" name="${id}management${childNum}-bgmanagementwork">
            </div>
            <div class="col form-group">
                <input type="number" class="form-control" id="${id}management${childNum}-bgmanagementperiod" name="${id}management${childNum}-bgmanagementperiod">
                <small id="${id}management${childNum}-bgmanagementperiod-label" class="form-text text-muted">숫자만 입력해주세요.</small>
            </div>
            <div class="col form-group">
                <div class="input-group">
                    <input type="number" class="form-control" id="${id}management${childNum}-bgmanagementratio" name="${id}management${childNum}-bgmanagementratio" step="0.01" max="100">
                    <div class="input-group-append">
                        <span class="input-group-text" style="background-color: #1C1C1C!important; color: #A7A59C!important; border: 1px solid #A7A59C!important;">%</span>
                    </div>
                </div>
            </div>
        </div>
        `
    }
    
    e.innerHTML = html;
    document.getElementById(id + "bgmanagement").appendChild(e);
    document.getElementById(id + "bgmanagementnum").value = document.getElementById(id + "bgmanagement").childElementCount;

    // 마지막 참여 기간을 제외한 다른 참여 기간들은 아래의 라벨을 지운다.
    document.getElementById(`${id}management${childNum - 1}-bgmanagementperiod-label`).innerHTML = "";
}

// addBGProjectPageBlankCheckFunc 함수는 예산 프로젝트 추가 페이지에서 Add 버튼을 눌렀을 때 빈칸을 확인하는 함수이다.
function addBGProjectPageBlankCheckFunc(type) {
    if (document.getElementById("id").value == "") {
        alert("Project ID를 입력해주세요");
        return false;
    }
    if (document.getElementById("name").value == "") {
        alert("프로젝트 한글명을 입력해주세요");
        return false;
    }
    if (document.getElementById("startdate").value == "") {
        alert("작업 예상 시작일을 입력해주세요");
        return false;
    }
    if (document.getElementById("enddate").value == "") {
        alert("작업 예상 마감일를 입력해주세요");
        return false;
    }
    var startdate = new Date(document.getElementById("startdate").value)
    var enddate = new Date(document.getElementById("enddate").value)
    if (startdate > enddate) {
        alert("작업 시작일과 작업 마감일이 잘못 입력되었습니다.\n다시 확인해주세요.")
        return false;
    }
    if (document.getElementById("status").value == "") {
        alert("예산 프로젝트의 Status를 선택하주세요");
        return false;
    }
    if (type == true) {
        if (document.getElementById("bgtype").value == "") {
            alert("최소한 예산안 타입의 이름은 입력해주세요");
            return false;
        }
    }
    if (type == false) {
        // 예산 프로젝트 수정 페이지에서 예산안 타입 중복 확인
        let tabList = [];
        let tabNum = Number(document.getElementById("tabnum").value);
        for (var i = 0; i < tabNum; i++) {
            let typeID = "type" + i + "-bgtype";
            if (document.getElementById(typeID) != null) {
                let typeName = document.getElementById(typeID).value.trim() // 앞뒤 공백 제거한 예산안 타입 이름
                if (typeName != "") {
                    if (tabList.includes(typeName)) {
                        alert(typeName + "과 동일한 이름의 예산안 타입이 존재합니다.");
                        return false;
                    }
                    else {
                        tabList.push(document.getElementById(typeID).value);
                    }
                }
                else {
                    alert("Tab" + i + "의 예산안 타입이 공백입니다. 확인해주세요.");
                    return false;
                }
            }
        }
    }
}

// addTabFunc 함수는 예산 프로젝트 수정 페이지에서 탭 추가
function addTabFunc(supervisor, production, management) {
    // 전달받은 수퍼바이저 아티스트 목록을 select에 넣어줄 옵션 형식으로 바꿔준다.
    let selectSUPHtml = "";
    let supArrayHtml = "";
    for (var i = 0; i < supervisor.length; i++) {
        selectSUPHtml += `<option value="${supervisor[i].ID}">${supervisor[i].Name}</option>\n`
        if (i == supervisor.length - 1) {
            supArrayHtml += `ID:${supervisor[i].ID},Name:${supervisor[i].Name}`
        } else {
            supArrayHtml += `ID:${supervisor[i].ID},Name:${supervisor[i].Name}/`
        }
    }
    
    // 전달받은 프로덕션 아티스트 목록을 select에 넣어줄 옵션 형식으로 바꿔준다.
    let selectPRODHtml = "";
    let prodArrayHtml = "";
    for (var i = 0; i < production.length; i++) {
        selectPRODHtml += `<option value="${production[i].ID}">${production[i].Name}</option>\n`
        if (i == production.length - 1) {
            prodArrayHtml += `ID:${production[i].ID},Name:${production[i].Name}`
        } else {
            prodArrayHtml += `ID:${production[i].ID},Name:${production[i].Name}/`
        }
    }

    // 전달받은 매니지먼트 아티스트 목록을 select에 넣어줄 옵션 형식으로 바꿔준다.
    let selectMNGHtml = "";
    let mngArrayHtml = "";
    for (var i = 0; i < management.length; i++) {
        selectMNGHtml += `<option value="${management[i].ID}">${management[i].Name}</option>\n`
        if (i == management.length - 1) {
            mngArrayHtml += `ID:${management[i].ID},Name:${management[i].Name}`
        } else {
            mngArrayHtml += `ID:${management[i].ID},Name:${management[i].Name}/`
        }
    }

    var tabID = Number(document.getElementById("tabnum").value)
    $('#tab-list').append($(
        `
        <li class="nav-item">
            <a class="nav-link" id="tab${tabID}" href="#type${tabID}" role="tab" data-toggle="tab">Tab${tabID}<button class="close" type="button" title="Remove this tab" style="padding-left: 10px;">x</button></a>
        </li>
        `
    ))
    $('#tab-content').append($(
        `
        <div class="tab-pane fade" id="type${tabID}">
            <div class="row">
                <div class="col">
                    <div class="row pt-3 pb-3">
                        <div class="ml-5">
                            <h5 class="section-heading text-muted">< 프로젝트 예산안 정보 ></h5>
                        </div>
                    </div>
                    <div class="row pt-3 pb-2">
                        <input type="hidden" name="type${tabID}-bgtypeid" value="">
                        <div class="col form-group">
                            <label class="text-muted">예산안 타입</label>
                            <input type="text" class="form-control" id="type${tabID}-bgtype" name="type${tabID}-bgtype" value="" onkeyup="changeTabNameFunc('type' + ${tabID} + '-bgtype');">
                        </div>
                        <div class="col form-group">
                            <label class="text-muted">계약일</label>
                            <input type="date" class="form-control" id="type${tabID}-bgcontractdate" name="type${tabID}-bgcontractdate" max="9999-12-31" value="">
                        </div>
                        <div class="col form-group" style="left: 30px;">
                            <div class="row">
                                <label class="text-muted">실행 예산 확정</label>
                            </div>
                            <div class="row pt-2">
                                <div class="custom-control custom-radio custom-control-inline" style="padding-left: 35px;">
                                    <input type="radio" id="type${tabID}-bgmaintypestatus1" name="type${tabID}-bgmaintypestatus" class="custom-control-input" value="true" onclick="offOtherStatusFunc(${tabID});">
                                    <label class="custom-control-label text-muted" for="type${tabID}-bgmaintypestatus1">Yes</label>
                                </div>
                                <div class="custom-control custom-radio custom-control-inline" style="padding-left: 35px;">
                                    <input type="radio" id="type${tabID}-bgmaintypestatus2" name="type${tabID}-bgmaintypestatus" class="custom-control-input" value="false">
                                    <label class="custom-control-label text-muted" for="type${tabID}-bgmaintypestatus2">No</label>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="row pb-2">
                        <div class="form-group col">
                            <label class="text-muted">제안 견적</label>
                            <input type="text" inputmode="numeric" class="form-control" id="type${tabID}-bgproposal" name="type${tabID}-bgproposal" value="" onkeyup="calNegoRatioFunc('type' + ${tabID});">
                            <small class="form-text text-muted">숫자만 입력해주세요.</small>
                        </div>
                        <div class="form-group col">
                            <label class="text-muted">계약 결정액</label>
                            <input type="text" inputmode="numeric" class="form-control" id="type${tabID}-bgdecision" name="type${tabID}-bgdecision" value="" onkeyup="calNegoRatioFunc('type' + ${tabID});">
                            <small class="form-text text-muted">숫자만 입력해주세요.</small>
                        </div>
                        <div class="form-group col">
                            <label class="text-muted">네고율</label>
                            <div class="input-group">
                                <input type="text" class="form-control" id="type${tabID}-bgnegoratio" name="type${tabID}-bgnegoratio" readonly>
                                <div class="input-group-append">
                                    <span class="input-group-text" style="background-color: #1C1C1C; color: #A7A59C; border: 1px solid #A7A59C;">%</span>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="row pb-2">
                        <div class="form-group col">
                            <label class="text-muted">계약 컷수</label>
                            <input type="text" inputmode="numeric" class="form-control" id="type${tabID}-bgcontractcuts" name="type${tabID}-bgcontractcuts" value="">
                            <small class="form-text text-muted">숫자만 입력해주세요.</small>
                        </div>
                        <div class="form-group col">
                            <label class="text-muted">작업 컷수</label>
                            <input type="text" inputmode="numeric" class="form-control" id="type${tabID}-bgworkingcuts" name="type${tabID}-bgworkingcuts" value="">
                            <small class="form-text text-muted">숫자만 입력해주세요.</small>
                        </div>
                        <div class="col"></div>
                    </div>
                    <div class="row pt-3 pb-3">
                        <div class="ml-5">
                            <h5 class="section-heading text-muted">< 예산 지출 기입 ></h5>
                        </div>
                    </div>
                    <div class="row pb-2">
                        <div class="form-group col">
                            <label class="text-muted">Retake율</label>
                            <div class="input-group">
                                <input type="number" class="form-control" id="type${tabID}-bgretakeratio" name="type${tabID}-bgretakeratio" step="0.01" max="100" value="">
                                <div class="input-group-append">
                                    <span class="input-group-text" style="background-color: #1C1C1C!important; color: #A7A59C!important; border: 1px solid #A7A59C!important;">%</span>
                                </div>
                            </div>
                        </div>
                        <div class="form-group col">
                            <label class="text-muted">진행비율</label>
                            <div class="input-group">
                                <input type="number" class="form-control" id="type${tabID}-bgprogressratio" name="type${tabID}-bgprogressratio" step="0.01" max="100" value="">
                                <div class="input-group-append">
                                    <span class="input-group-text" style="background-color: #1C1C1C!important; color: #A7A59C!important; border: 1px solid #A7A59C!important;">%</span>
                                </div>
                            </div>
                        </div>
                        <div class="form-group col">
                            <label class="text-muted">외주비율</label>
                            <div class="input-group">
                                <input type="number" class="form-control" id="type${tabID}-bgvendorratio" name="type${tabID}-bgvendorratio" step="0.01" max="100" value="">
                                <div class="input-group-append">
                                    <span class="input-group-text" style="background-color: #1C1C1C!important; color: #A7A59C!important; border: 1px solid #A7A59C!important;">%</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="col-sm-1"></div>
                <div class="col">
                    <div class="row pt-3 pb-3">
                        <div class="col-sm-9">
                            <div class="ml-5">
                                <h5 class="section-heading text-muted">< 수퍼바이저 / 프로덕션 / 매니지먼트 기입 ></h5>
                            </div>
                        </div>
                        <div class="col-sm-1 mt-1">
                            <input type="hidden" name="type${tabID}-bgsupervisornum" id="type${tabID}-bgsupervisornum" value="1">
                            <span id="type${tabID}-bgsupervisoraddbtn" class="add float-right" onclick="addBGSupervisorSettingFunc('${supArrayHtml}', 'type${tabID}');">SUP</span>
                        </div>
                        <div class="col-sm-1 mt-1">
                            <input type="hidden" name="type${tabID}-bgproductionnum" id="type${tabID}-bgproductionnum" value="1">
                            <span id="type${tabID}-bgproductionaddbtn" class="add float-right" onclick="addBGProductionSettingFunc('${prodArrayHtml}', 'type${tabID}');">PROD</span>
                        </div>
                        <div class="col-sm-1 mt-1">
                            <input type="hidden" name="type${tabID}-bgmanagementnum" id="type${tabID}-bgmanagementnum" value="1">
                            <span id="type${tabID}-bgmanagementaddbtn" class="add float-right" onclick="addBGManagementSettingFunc('${mngArrayHtml}', 'type${tabID}');">MNG</span>
                        </div>
                    </div>
                    <div id="type${tabID}-bgsupervisor">
                        <div class="row pt-3 pb-1">
                            <div class="col form-group">
                                <label class="text-muted">수퍼바이저</label>
                                <select name="type${tabID}-supervisor0" id="type${tabID}-supervisor0" class="form-control custom-select left-radius right-radius">
                                    <option value=""></option>
                                    ${selectSUPHtml}
                                </select>
                            </div>
                            <div class="col form-group">
                                <label class="text-muted">업무</label>
                                <input type="text" class="form-control" id="type${tabID}-supervisor0-bgmanagementwork" name="type${tabID}-supervisor0-bgmanagementwork" value="">
                            </div>
                            <div class="col form-group">
                                <label class="text-muted">참여 기간</label>
                                <input type="number" class="form-control" id="type${tabID}-supervisor0-bgmanagementperiod" name="type${tabID}-supervisor0-bgmanagementperiod" value="">
                                <small class="form-text text-muted">숫자만 입력해주세요.</small>
                            </div>
                            <div class="col form-group">
                                <label class="text-muted">참여 기간 퍼센티지</label>
                                <div class="input-group">
                                    <input type="number" class="form-control" id="type${tabID}-supervisor0-bgmanagementratio" name="type${tabID}-supervisor0-bgmanagementratio" step="0.01" max="100" value="">
                                    <div class="input-group-append">
                                        <span class="input-group-text" style="background-color: #1C1C1C!important; color: #A7A59C!important; border: 1px solid #A7A59C!important;">%</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div id="type${tabID}-bgproduction">
                        <div class="row pt-5 pb-1">
                            <div class="col form-group">
                                <label class="text-muted">프로덕션</label>
                                <select name="type${tabID}-production0" id="type${tabID}-production0" class="form-control custom-select left-radius right-radius">
                                    <option value=""></option>
                                    ${selectPRODHtml}
                                </select>
                            </div>
                            <div class="col form-group">
                                <label class="text-muted">업무</label>
                                <input type="text" class="form-control" id="type${tabID}-production0-bgmanagementwork" name="type${tabID}-production0-bgmanagementwork" value="">
                            </div>
                            <div class="col form-group">
                                <label class="text-muted">참여 기간</label>
                                <input type="number" class="form-control" id="type${tabID}-production0-bgmanagementperiod" name="type${tabID}-production0-bgmanagementperiod" value="">
                                <small class="form-text text-muted">숫자만 입력해주세요.</small>
                            </div>
                            <div class="col form-group">
                                <label class="text-muted">참여 기간 퍼센티지</label>
                                <div class="input-group">
                                    <input type="number" class="form-control" id="type${tabID}-production0-bgmanagementratio" name="type${tabID}-production0-bgmanagementratio" step="0.01" max="100" value="">
                                    <div class="input-group-append">
                                        <span class="input-group-text" style="background-color: #1C1C1C!important; color: #A7A59C!important; border: 1px solid #A7A59C!important;">%</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div id="type${tabID}-bgmanagement">
                        <div class="row pt-5 pb-1">
                            <div class="col form-group">
                                <label class="text-muted">매니지먼트</label>
                                <select name="type${tabID}-management0" id="type${tabID}-management0" class="form-control custom-select left-radius right-radius">
                                    <option value=""></option>
                                    ${selectMNGHtml}
                                </select>
                            </div>
                            <div class="col form-group">
                                <label class="text-muted">업무</label>
                                <input type="text" class="form-control" id="type${tabID}-management0-bgmanagementwork" name="type${tabID}-management0-bgmanagementwork" value="">
                            </div>
                            <div class="col form-group">
                                <label class="text-muted">참여 기간</label>
                                <input type="number" class="form-control" id="type${tabID}-management0-bgmanagementperiod" name="type${tabID}-management0-bgmanagementperiod" value="">
                                <small class="form-text text-muted">숫자만 입력해주세요.</small>
                            </div>
                            <div class="col form-group">
                                <label class="text-muted">참여 기간 퍼센티지</label>
                                <div class="input-group">
                                    <input type="number" class="form-control" id="type${tabID}-management0-bgmanagementratio" name="type${tabID}-management0-bgmanagementratio" step="0.01" max="100" value="">
                                    <div class="input-group-append">
                                        <span class="input-group-text" style="background-color: #1C1C1C!important; color: #A7A59C!important; border: 1px solid #A7A59C!important;">%</span>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        `
    ))

    document.getElementById("tabnum").value = tabID + 1
}

// offOtherStatusFunc 함수는 실행 예산 확정 버튼에서 Yes를 눌렀을 때 다른 실행 예산을 No로 바꾸는 함수이다.
function offOtherStatusFunc(typeNum) {
    let tabNum = Number(document.getElementById("tabnum").value)
    for (let i = 0; i < tabNum; i++) {
        if (i == Number(typeNum)) {
            continue
        }
        if (document.getElementById("type" + i + "-bgmaintypestatus2") != null) {
            document.getElementById("type" + i + "-bgmaintypestatus2").checked = true
        }
    }
}

// changeTabNameFunc 함수는 예산안 타입 입력이 바뀜에 따라 해당하는 탭의 이름이 바뀌는 함수이다.
function changeTabNameFunc(id) {
    let typeName = document.getElementById(id).value
    id = "tab" + id.split("-")[0].replace("type", "")
    document.getElementById(id).innerHTML = `${typeName}<button class="close" type="button" title="Remove this tab" style="padding-left: 10px;">x</button>`
}

/* 기타 함수 */
// getPositionOfCursorFunc 함수는 커서의 위치를 가져오는 함수이다.
function getPositionOfCursorFunc(tag) {
    var position = { start: 0, end: 0 };

    if ( tag.selectionStart) {
        position.start = tag.selectionStart;
        position.end = tag.selectionEnd;
    }

    return position
}

// 금액 입력 칸에 세자리마다 콤마 찍히는 기능
$(document).on('keyup','input[inputmode=numeric]', function(event){

    // "shift키 + 방향키"를 눌렀을 때에는 제외시키기
    event = event || window.event;
    var keyCode = event.which || event.keyCode;
    if (keyCode == 16 || (36 < keyCode && keyCode <41 )) return;
    if (keyCode == 17 || keyCode == 65) return; // ctrl + a 키 눌렀을 때에는 제외시키기

    var cursor = getPositionOfCursorFunc(this); // 커서의 위치 가져오기
    var beforeLength = this.value.length; // 원래 텍스트의 전체 길이
    this.value = this.value.replace(/[^0-9]/g,''); // 입력값이 숫자가 아니면 공백
    this.value = this.value.replace(/,/g,''); // ,값 공백처리
    this.value = this.value.replace(/\B(?=(\d{3})+(?!\d))/g, ","); // 정규식을 이용해서 3자리 마다 , 추가
    var afterLength = this.value.length; // 바뀐 텍스트의 전체 길이
    var gap = afterLength - beforeLength;

    // 커서의 위치 바꾸기
    if (this.selectionStart) {
        this.selectionStart = cursor.start + gap;
        this.selectionEnd = cursor.end + gap;
    } else if (this.createTextRange) {
        var start = cursor.start - beforeLength;
        var end = cursor.end - beforeLength;

        var range = this.createTextRange();

        range.collapse(false);
        range.moveStart("character", start);
        range.moveEnd("character", end);
        range.select();
    }
});