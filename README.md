# 프로젝트 재무관리 툴
프로젝트 작업 비용 + 외주 비용 + 기타 비용들을 합산한 총 내역을 확인할 수 있는 툴

<br>

## 결산 툴 세팅
#### Mongo DB 설치 및 실행
###### 설치
```bash
$ sudo /rd/land/OPT/TD/pcsetting/linux/installer/mongo.sh
```

###### 암호 설정
```bash
# mongoDB에 사용자 추가
$ mongo
> use admin
switched to db admin

> db.createUser(
... {
... user: "rdadmin",  # admin 계정은 이미 존재하기 때문에 admin를 제외한 ID 입력
... pwd: passwordPrompt(),
... roles: [ { role: "userAdminAnyDatabase", db: "admin" }, "readWriteAnyDatabase" ]
... }
... )
Enter password: 

Successfully added user: {
	"user" : "rdadmin",
	"roles" : [
		{
			"role" : "userAdminAnyDatabase",
			"db" : "admin"
		},
		"readWriteAnyDatabase"
	]
}

# mongod.conf 파일에서 security 부분을 아래와 같이 수정
$ vim /etc/mongod.conf

    security:
      authorization: enabled

# mongod 서비스 재시작
$ systemctl restart mongod

# 연결 확인
$ mongo
> use admin
switched to db admin

> db.auth("rdadmin", passwordPrompt())
Enter password: 
1  # 0: 인증 실패, 1: 인증 성공

> use budget
switched to db budget
```

<br>

#### 초기 설정
###### 암호화 키 생성
```bash
$ sudo budget -gen-key
```

###### admin 계정 생성
```bash
$ sudo budget -add user -id admin -password password -name 관리자 -team 관리자 -accesslevel 4
```

###### 포트 열기
```bash
$ sudo firewall-cmd --permanent --zone=public --add-port=80/tcp
$ sudo firewall-cmd --reload
```

<br>


#### Budget 실행
```bash
# 같은 서버에 있는 DB를 사용할 경우
$ sudo budget -http :80

# 다른 서버에 있는 DB를 사용할 경우
$ sudo budget -http :80 -mongodburi mongodb://10.20.30.45:27017

```

10.20.30.192 MAC address (고정): 52:54:00:df:6a:e9   
10.20.31.160 MAC address (테스트 - 애림): b4:2e:99:6e:a1:07

<br>

## 매뉴얼
#### Command Line
- [유저](docs/cmd_user.md)
- [아티스트](docs/cmd_artist.md)
- [타임로그](docs/cmd_timelog.md)
- [Admin Setting](docs/cmd_adminsetting.md)
- [프로젝트](docs/cmd_project.md)
- [Vendor](docs/cmd_vendor.md)

<br>

#### Rest API
- [아티스트](docs/restapi_artist.md)
- [사용자](docs/restapi_user.md)
- [Shotgun](docs/restapi_shotgun.md)
- [Admin Setting](docs/restapi_adminsetting.md)
- [Timelog](docs/restapi_timelog.md)