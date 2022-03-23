import os
import requests
import json
import sys

token =''
source_dir = os.environ['source_dir']
def get_token():
  sirvurl = 'https://api.sirv.com/v2/token'
  payload = {
      'clientId': os.environ['clientId'],
      'clientSecret': os.environ['clientSecret']
  }
  headers = {'content-type': 'application/json'}
  response = requests.request('POST', sirvurl, data=json.dumps(payload), headers=headers)
  global token
  if response:
    token = response.json()['token']
  else:
    print('There is an error in your credentials. Please check if your ID and secret are correct.')
    sys.exit()

def check_folder():
  get_token()
  sirvurl = 'https://api.sirv.com/v2/files/readdir?dirname=/docs/'
  headers = {
      'content-type': 'application/json',
      'authorization': 'Bearer %s' % token
  }
  response = requests.request('GET', sirvurl, headers=headers)
  status_code = response.status_code
  if status_code == 200:
    fetch_files()
  else:
    create_folder()
    fetch_files()


def create_folder():
  sirvurl = 'https://api.sirv.com/v2/files/mkdir?dirname=/docs/'
  headers = {
      'content-type': 'application/json',
      'authorization': 'Bearer %s' % token
  }
  requests.request('POST', sirvurl, headers=headers)

def fetch_files():
  directory = source_dir
  for filename in os.listdir(directory):
      f = os.path.join(directory, filename)
      if os.path.isfile(f):
          print(filename)
          get_token()
          sirvurl = 'https://api.sirv.com/v2/files/upload'
          qs = {'filename':  '/docs/'+filename}
          payload = open(f, 'rb')

          headers = {
              'content-type': 'image/jpeg',
              'authorization': 'Bearer %s' % token
          }
          try:
            response = requests.request('POST', sirvurl, data=payload, headers=headers, params=qs)
            print(response)
          except requests.exceptions.RequestException as e:
            raise SystemExit(e)

check_folder()