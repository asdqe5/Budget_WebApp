# Project
Project 관련 터미널 명령어 사용법입니다.

<br>

##### 프로젝트 추가
DB에 프로젝트를 추가합니다. 프로젝트 id, 이름, 작업기간 시작과 끝, 총매출(계약금)을 기입해줘야합니다. 이때 프로젝트 id는 DB에 대문자로 저장됩니다.   
```bash
$ budget -add project -id bee -name 강철비 -startdate 2020-03 -enddate 2020-08 -payment 10000000
```

이미 정산 완료된 프로젝트를 추가할 경우, isfinished를 true로 입력하고 총 내부비용을 필수로, 세부사항은 부가적으로 입력해주면 됩니다.
```bash
$ budget -add project -id bee -name 강철비 -startdate 2020-03 -enddate 2020-08 -payment 10000000 -totalamount 5000000 -laborcost 1000000 -progresscost 1000000 -purchasecost 1000000 -isfinished true

# totalcost(총내부비용), laborcost(내부인건비), progresscost(진행비), purchasecost(구매비)
```

<br>

##### 프로젝트 삭제
DB에서 프로젝트를 검색하여 삭제합니다. 프로젝트 id만 입력합니다. 
```bash
$ budget -rm project -id bee
```

<br>

##### 프로젝트 가져오기
DB에서 프로젝트 id가 일치하는 프로젝트를 검색하여 출력해줍니다.
```bash
$ budget -get project -id bee
```

<br>

##### 프로젝트 검색하기
DB에서 검색어를 이용하여 프로젝트를 검색하여 출력해줍니다.  
사용할 수 있는 키워드 : id, name, date
```bash
$ budget -search project -id bee
$ budget -search project -name 강철비
$ budget -search project -date 2020-05 # 작업기간 중인 프로젝트를 모두 검색하여 출력해줍니다.
```

<br>

##### 프로젝트 설정(수정)하기
DB에 있는 프로젝트를 검색하여 해당 프로젝트의 이름, 총매출(계약금)을 입력받아 수정합니다.
```bash
$ budget -set project -id bee -name 강철비2 -payment 20000000
```

