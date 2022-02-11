# User
유저 관련 터미널 명령어 사용법입니다.

<br>

##### 유저 추가
DB에 유저를 추가합니다. ID, Password, 이름, 팀을 입력해줘야합니다. Accesslevel의 경우 따로 적어주지 않으면 GusetLevel로 추가됩니다.

    ```bash
    $ budget -add user -id admin -password admin -name 김준섭 -team admin
    ```
AccessLevel을 입력할 경우 이에 맞게 유저가 추가됩니다.
    
    ```bash
    $ budget -add user -id admin -password admin -name 김준섭 -team admin -accesslevel 3
    ```