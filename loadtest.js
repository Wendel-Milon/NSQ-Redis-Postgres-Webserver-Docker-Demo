import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 250 },
    // { duration: '1m30s', target: 10 },
    // { duration: '20s', target: 0 },
  ],
};

// Load test the protected route.
// export default function () {
//   const res = http.get('http://localhost:8080/protected', {
//     cookies: {
//         csrftoken: "78b5a6c4-80cf-4146-b363-2c7068644bc3",
//     }
//   });
//   check(res, { 'status was 200': (r) => r.status == 200 });
// }


export default function () {
    let res = http.get('http://localhost:8080/login');
    res = res.submitForm({

    formSelector: 'form',

    fields: { userid: 'test', passwd: 'test' },
  });
    check(res, { 'status was 200': (r) => r.status == 200});
  }