import requests

user_login = "test"
pwd = "1"
session = requests.Session()
token = ""


def register():
    try:
        responce = session.post("http://localhost:8080/api/user/register", 
                                json={"login": user_login, "password": pwd})
        print(f"REGISTER: {responce.status_code}")
    except Exception as ex:
        print(f"Registration error: {ex}", flush=True)

def login():
    global token
    try:
        responce = session.post("http://localhost:8080/api/user/login", 
                                json={"login": user_login, "password": pwd})
        token = responce.headers.get("Authorization", "")
        print(f"LOGIN: {responce.status_code}, {token   }")
    except Exception as ex:
        print(f"LOGIN error: {ex}", flush=True)

def addOrder():
    global token
    try:
        responce = session.post("http://localhost:8080/api/user/orders", 
                                data="55875248746", headers={"Authorization": token})
        print(f"ADD ORDER: {responce.status_code}")
    except Exception as ex:
        print(f"ADD ORDER error: {ex}", flush=True)
        
def getOrder():
    global token
    try:
        responce = session.get("http://localhost:8080/api/user/orders", headers={"Authorization": token})
        print(f"GET ORDER: {responce.status_code} {responce.text}")
    except Exception as ex:
        print(f"GET ORDER error: {ex}", flush=True)     
           
def getBalance():
    global token
    try:
        responce = session.get("http://localhost:8080/api/user/balance", headers={"Authorization": token})
        print(f"GET BALANCE: {responce.status_code} {responce.text}")
    except Exception as ex:
        print(f"GET BALANCE error: {ex}", flush=True)
        
def addWithdraw():
    global token
    try:
        responce = session.post("http://localhost:8080/api/user/balance/withdraw",
                                json={"order": "2377225624", "sum": 100.1}, 
                                headers={"Authorization": token})
        print(f"ADD withdraw: {responce.status_code} {responce.text}")
    except Exception as ex:
        print(f"ADD withdraw error: {ex}", flush=True)
        
def getWithdraws():
    global token
    try:
        responce = session.get("http://localhost:8080/api/user/withdrawals", 
                                headers={"Authorization": token})
        print(f"GET withdraws: {responce.status_code} {responce.text}")
    except Exception as ex:
        print(f"GET withdraws error: {ex}", flush=True)
        
        

register()
login()
addOrder()
getOrder()
getBalance()
addWithdraw()
getWithdraws()