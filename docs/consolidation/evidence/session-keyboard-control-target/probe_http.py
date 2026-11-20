#!/usr/bin/env python3
"""Bounded HTTP reproduction using an existing exact-source binary; no browser/build."""
import hashlib
from html.parser import HTMLParser
import http.cookiejar
import json
import os
from pathlib import Path
import re
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request

OUT = Path(__file__).resolve().parent
BINARY = Path('/tmp/circula-ui-e0')

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None

class Client:
    def __init__(self, base):
        self.base = base
        self.jar = http.cookiejar.CookieJar()
        self.opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.jar), NoRedirect)

    def request(self, path, form=None):
        data = urllib.parse.urlencode(form).encode() if form is not None else None
        request = urllib.request.Request(self.base + path, data=data)
        try:
            response = self.opener.open(request, timeout=10)
        except urllib.error.HTTPError as response_error:
            response = response_error
        with response:
            return response.status, response.headers, response.read().decode()

    def cookies(self):
        return {c.name: c.value for c in self.jar}

def csrf(body):
    return re.search(r'name="csrf" value="([^"]+)"', body).group(1)

def login(client, name):
    status, _, body = client.request('/login')
    assert status == 200
    status, _, _ = client.request('/login', {'login':name, 'password':'correct horse battery staple', 'csrf':csrf(body)})
    assert status == 303, status

class Controls(HTMLParser):
    """Capture HTML control candidates and label text in DOM order; no layout engine."""
    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.controls = []
        self.labels = {}
        self.label = None
        self.text_control = None
        self.form = None
        self.form_count = 0

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if tag == 'form':
            self.form_count += 1
            self.form = self.form_count
        if tag == 'label':
            self.label = {'for':attrs.get('for'), 'text':''}
        if ((tag == 'a' and 'href' in attrs) or tag in ('button','textarea','select')
                or tag == 'input' and attrs.get('type','text') != 'hidden') and 'disabled' not in attrs:
            # Remove dynamic, noncredential session selectors from captured routes.
            attrs = {k:re.sub(r'/account/sessions/[0-9a-f]{64}/revoke', '/account/sessions/fixture/revoke', v)
                     if isinstance(v,str) else v for k,v in attrs.items()}
            control = {'tagName':tag.upper(), 'type':attrs.get('type','text') if tag == 'input' else '',
                       'attributes':attrs, 'text':'', 'form':self.form}
            self.controls.append(control)
            if tag in ('a','button','textarea','select'):
                self.text_control = control

    def handle_endtag(self, tag):
        if tag == 'form':
            self.form = None
        if tag == 'label' and self.label:
            self.labels[self.label['for']] = ' '.join(self.label['text'].split())
            self.label = None
        if self.text_control and tag.upper() == self.text_control['tagName']:
            self.text_control = None

    def handle_data(self, text):
        if self.label:
            self.label['text'] += text
        if self.text_control:
            self.text_control['text'] += text

    def result(self):
        for c in self.controls:
            c['text'] = ' '.join(c['text'].split())
            c['label'] = self.labels.get(c['attributes'].get('id'),'')
        return self.controls

def controls(body):
    p = Controls()
    p.feed(body)
    return p.result()

started = time.monotonic()
with tempfile.TemporaryDirectory(prefix='circula-session-http-') as tmp:
    with socket.socket() as listener:
        listener.bind(('127.0.0.1',0))
        port = listener.getsockname()[1]
    base = f'http://127.0.0.1:{port}'
    env = {'PATH':os.environ['PATH'], 'GOMAXPROCS':'1', 'GOMEMLIMIT':'250MiB', 'GOGC':'40',
           'CIRCULA_LISTEN':f'127.0.0.1:{port}', 'CIRCULA_PUBLIC_BASE_URL':base,
           'CIRCULA_DATABASE':tmp+'/circula.db','CIRCULA_BACKUP_DIR':tmp+'/backups',
           'CIRCULA_BACKUP_INTERVAL':'0','CIRCULA_MIGRATE_MODE':'auto',
           'CIRCULA_COOKIE_SECURE':'false','CIRCULA_LOG_FORMAT':'text',
           'CIRCULA_SETUP_SECRET':'e2e-setup-secret-0123456789abcdef'}
    with open(tmp+'/server.log','wb') as server_log:
        child = subprocess.Popen([str(BINARY),'serve'],env=env,stdout=server_log,stderr=server_log)
        try:
            owner = Client(base)
            for _ in range(100):
                assert child.poll() is None, 'server exited'
                try:
                    if owner.request('/healthz')[0] == 200:
                        break
                except urllib.error.URLError:
                    time.sleep(0.05)
            else:
                raise AssertionError('server readiness deadline exceeded')
            name = 'session-fixture@example.test'
            status, _, body = owner.request('/register')
            assert status == 200
            status, _, _ = owner.request('/register',{'login':name,'password':'correct horse battery staple',
                    'password2':'correct horse battery staple','csrf':csrf(body)})
            assert status == 303
            login(owner,name)
            other = Client(base)
            login(other,name)
            status, _, listing = other.request('/account/sessions')
            assert status == 200
            match = re.search(r'<a href="(/account/sessions/[0-9a-f]{64}/revoke)">Sign out here</a>',listing)
            assert match, 'other session link missing'
            target = match.group(1)
            status, _, body = owner.request(target)
            assert status == 200 and 'You will stay signed in here.' in body
            token = csrf(body)
            before = owner.cookies()
            invalid_status, invalid_headers, invalid_body = owner.request(target,{'confirm':'yes','csrf':'invalid'})
            after = owner.cookies()
            assert invalid_status == 403
            assert before['circula_session'] == after['circula_session']
            assert before['cm_csrf'] == after['cm_csrf']
            assert csrf(invalid_body) == token
            missing_status, _, missing_body = owner.request(target,{'csrf':token})
            assert missing_status == 422
            assert csrf(missing_body) == token
            dom = controls(missing_body)
            text = 'I want to end this session.'
            matches = [(i,c) for i,c in enumerate(dom) if (c['label'] or c['text']) == text]
            assert len(matches) == 2
            assert matches[0][1]['tagName'] == 'A'
            assert matches[0][1]['attributes']['href'] == '#f-confirm'
            assert matches[1][1]['tagName'] == 'INPUT' and matches[1][1]['type'] == 'checkbox'
            assert 'checked' not in matches[1][1]['attributes']
            # The proposed reload happens before the intentional validation error.
            reload_status, _, reloaded = owner.request(target)
            assert reload_status == 200 and csrf(reloaded) == token
            missing_again, _, repeated_body = owner.request(target,{'csrf':token})
            assert missing_again == 422
            assert controls(repeated_body) == dom
            positive_status, positive_headers, _ = owner.request(target,{'confirm':'yes','csrf':token})
            assert positive_status == 303 and positive_headers['Location'] == '/account/sessions'
            other_status = other.request('/api/v1/me/sessions')[0]
            current_status = owner.request('/api/v1/me/sessions')[0]
            assert other_status == 401 and current_status == 200
            fixture = {'provenance':{'source':'e0d27cc7cb73299219439919b260991b824d594a',
                        'method':'Actual HTTP session-revocation validation response, status422; dynamic target selector replaced with fixture',
                        'purpose':'HTML control candidates in DOM order, including hidden ancestors; captured by HTMLParser, without browser layout or navigation'},
                       'controls':dom}
            (OUT/'session-revoke-error-controls.json').write_text(json.dumps(fixture,indent=2)+'\n')
            result = {'source':'e0d27cc7cb73299219439919b260991b824d594a','binary':str(BINARY),
                      'binary_sha256':hashlib.sha256(BINARY.read_bytes()).hexdigest(),
                      'invalid_csrf_status':invalid_status,'session_cookie_unchanged':True,
                      'csrf_cookie_unchanged':True,'original_form_token_unchanged':True,
                      'invalid_response_auth_cookie_names':[h.split('=',1)[0] for h in invalid_headers.get_all('Set-Cookie',[])],
                      'missing_confirmation_status':missing_status,
                      'same_name_control_indices':[i for i,_ in matches],
                      'same_name_control_types':['link','checkbox'],
                      'reload_before_missing_confirmation_preserves_collision':True,
                      'repeated_missing_confirmation_status':missing_again,
                      'confirmed_post_with_original_token_status':positive_status,
                      'confirmed_post_location':positive_headers['Location'],
                      'target_session_api_status':other_status,'current_session_api_status':current_status,
                      'runtime_seconds':round(time.monotonic()-started,3),
                      'scope':'HTTP and rendered control capture only; no browser, Go build or production change'}
            (OUT/'http-probe-result.json').write_text(json.dumps(result,indent=2)+'\n')
            print(json.dumps(result,indent=2))
        finally:
            child.terminate()
            try:
                child.wait(timeout=5)
            except subprocess.TimeoutExpired:
                child.kill()
                child.wait()
