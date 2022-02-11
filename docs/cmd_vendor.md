# Vendor
Vendor 관련 터미널 명령어 사용법입니다.

<br>

##### Vendor 추가
DB에 Vendor를 추가합니다. 프로젝트를 기준으로 저장을 하며, DB에 프로젝트가 존재하지 않는 경우 추가되지 않습니다.
프로젝트 ID, 벤더명, 총 비용은 필수로 추가를 해줘야합니다. 이때 프로젝트 ID는 대문자로 저장됩니다.
```bash
$ budget -add vendor -project bec -name 유르테 -startdate 2020-12-04 -enddate 2021-02-23 -payment 10000000

# 컷 정보를 입력할 경우 -cuts 30 -tasks fx,comp,lighting
```

##### Vendor 검색
DB에 존재하는 Vendor를 검색합니다. 프로젝트ID 및 벤더명으로 검색할 수 있습니다,
```bash
$ budget -search vendor -project bee -name 벙커 # 프로젝트 ID, 벤더명 둘 중에 하나는 필수로 입력해야 합니다.
```

##### Vendor 삭제
DB에 존재하는 Vendor를 삭제합니다. 삭제하고자하는 프로젝트ID와 벤더명을 모두 입력해야합니다.
```bash
$ budget -rm vendor -project bee -name 벙커
```