# Artist
아티스트 관련 터미널 명령어 사용법입니다.

<br>

##### 아티스트 추가
###### VFX팀
DB에 VFX팀 아티스트를 추가합니다. 부서, 팀, 이름을 따로 입력할 필요없이 id만 입력해주면 Shotgun에서 이름, 부서, 팀 정보를 가져와 자동으로 기입해줍니다.
    ```bash
    $ budget -add artistvfx -id 90 -salary "2020:2400,2019:2300"
    ```

###### CM팀
DB에 CM팀 아티스트를 추가합니다. CM팀의 경우 id, 이름, 팀, 연봉을 모두 기입해줘야합니다.
이때 부서는 자동으로 'cm'으로, id는 'cm'이 붙어서 DB에 저장됩니다.
    ```bash
    $ budget -add artistcm -id 999 -name 김준섭 -team vfx -salary "2020:2400,2019:2300"
    ```

<br>

##### 아티스트 삭제
DB에서 아티스트를 검색하여 삭제합니다.
```bash
$ budget -rm artist -id 90
```

<br>

##### 아티스트 가져오기
DB에서 ID가 일치하는 아티스트를 검색하여 출력해줍니다.
```bash
$ budget -get artist -id 90
```

<br>

##### 아티스트 검색하기
DB에서 검색어를 이용하여 아티스트를 검색하여 출력해줍니다.  
사용할 수 있는 키워드 : id, name, dept, team
```bash
$ budget -search artist -id 90                 # ID가 90인 아티스트 출력
$ budget -search artist -dept comp -team Comp1 # comp 부서의 Comp1 팀인 아티스트 출력
```

<br>

##### 아티스트 퇴사 여부 설정
아티스트의 퇴사일이 설정되어 있으면, 오늘 날짜랑 비교하여 퇴사 여부를 true로 설정한다.
```bash
$ budget -set-resination
```