# Timelog
타임로그 관련 터미널 명령어 사용법입니다.

<br>

##### 타임로그 추가
DB에 타임로그를 추가합니다. 만약 아티스트, 날짜, 프로젝트 정보가 일치하는 타임로그가 존재하면 duration이 업데이트됩니다.
```bash
$ budget -add timelog -id 90 -year 2020 -month 7 -project bee -duration 8
```

<br>

##### 타임로그 삭제
DB에서 타임로그를 검색하여 삭제합니다.
```bash
$ budget -rm timelog -id 90 -year 2020 -month 8 -project bee
```

<br>

##### 타임로그 duration 빼기
DB에서 아티스트, 날짜, 프로젝트 정보가 일치하는 타임로그를 검색하여 duration을 빼줍니다. 만약 저장된 duration보다 큰 숫자를 입력하면 에러가 발생하고, 빼서 0이 되면 DB에서 삭제됩니다.
```bash
$ budget -sub-timelog -id 90 -year 2020 -month 7 -project bee -duration 8
```

<br>

##### 타임로그 가져오기
DB에서 아티스트, 날짜, 프로젝트 정보가 일치하는 타임로그를 검색하여 출력해줍니다.
```bash
$ budget -get timelog -id 90 -year 2020 -month 7 -project bee
```

<br>

##### 타임로그 검색하기
DB에서 검색어를 이용하여 타임로그를 검색하여 출력해줍니다.  
사용할 수 있는 키워드 : id, quarter, year, month, project
```bash
$ budget -search timelog -id 90            # ID가 90인 아티스트가 작성한 타임로그 출력
$ budget -search timelog -id 90 -year 2020 # 2020년에 ID가 90인 아티스트가 작성한 타임로그 출력
```

<br>

##### 타임로그 업데이트
월별 결산 상태를 확인한 후에 현재 달의 타임로그 혹은 지난 달과 현재 달의 타임로그를 DB에 업데이트합니다.
데이터의 타임로그를 모두 계산하여 DB에 업데이트합니다.
```bash
$ budget -update-timelog
```

<br>

##### 타임로그 리셋
샷건에 저장되어 있는 모드 타임로그 데이터를 DB에 리셋합니다.
이때, 데이터의 모든 타임로그를 계산하여 DB에 업데이트합니다.
```bash
$ budget -reset-timelog
```

