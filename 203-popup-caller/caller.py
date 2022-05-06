import os
from flask import Flask, request, abort
from twilio.rest import Client

app = Flask(__name__)

account_sid = os.environ['TWILIO_ACCOUNT_SID']
if not account_sid:
    print('Please set TWILIO_ACCOUNT_SID')
    os.exit(1)

auth_token = os.environ['TWILIO_AUTH_TOKEN']
if not auth_token:
    print('Please set TWILIO_AUTH_TOKEN')
    os.exit(1)

from_number = os.environ['TWILIO_FROM_NUMBER']
if not from_number:
    print('Please set TWILIO_FROM_NUMBER')
    os.exit(1)

to_number = os.environ['TWILIO_TO_NUMBER']
if not to_number:
    print('Please set TWILIO_TO_NUMBER')
    os.exit(1)

may_i = os.environ['MAY_I_ANSWER']
if not to_number:
    print('Please set MAY_I_ANSWER')
    os.exit(1)

twilioc = Client(account_sid, auth_token)

@app.route('/callmom', methods=['POST'])
def callmom():
    if request.get_json(force=True).get('may_i') != may_i:
        return abort(403)

    twilioc.calls.create(
        url='https://handler.twilio.com/twiml/EHca3729a5f618dc0a1f247d78f36bfd8c',
        to=to_number,
        from_=from_number,
    )
    return 'OK'
