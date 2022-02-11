# Admin Setting
Admin Setting 관련 터미널 명령어 사용법입니다.

<br>

##### 월별 결산 상태 설정
DB에 월별 결산 상태를 추가합니다. DB에 데이터가 없으면 추가하고, 있으면 status를 업데이트합니다.
    ```bash
    $ budget -set monthlystatus -date "2020-09" -status true
    ```
