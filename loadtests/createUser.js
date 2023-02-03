import http from 'k6/http';
import { check, sleep } from 'k6';
import {uuidv4} from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

export const options = {
  stages: [
    { duration: '60s', target: 2000 },
  ],
};


export default function () {
    let res = http.get('http://localhost:8080/create');
    res = res.submitForm({

    formSelector: 'form',
    fields: { userid: uuidv4(), passwd: 'test' },
  });
    check(res, { 'status was 200': (r) => r.status == 200});
  }

