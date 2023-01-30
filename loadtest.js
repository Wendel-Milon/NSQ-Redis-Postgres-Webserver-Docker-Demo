import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '60s', target: 5000 },
    // { duration: '1m30s', target: 10 },
    // { duration: '20s', target: 0 },
  ],
};

// Load test the protected route.
export default function () {
  const res = http.get('http://localhost:8080', {});
  //   cookies: {
  //       csrftoken: "78b5a6c4-80cf-4146-b363-2c7068644bc3",
  //   }
  // });
  check(res, { 'status was 200': (r) => r.status == 200 });
}


// export default function () {
//     let res = http.get('http://localhost:8080/login');
//     res = res.submitForm({

//     formSelector: 'form',

//     fields: { userid: 'test', passwd: 'test' },
//   });
//     check(res, { 'status was 200': (r) => r.status == 200});
//   }


  // export default function () {
  //   let res = http.post('http://localhost:8080/protected',{
  //         cookies: {
  //             csrftoken: "6ae5a2c9-2de6-4b97-b6e7-7bb900ad05df",
  //         }
  //       });
  //   check(res, { 'status was 200': (r) => r.status == 200});
  // }