import { openSync } from 'k6/experimental/fs';

export const options = {
    scenarios: {
      default: {
        executor: 'constant-vus',
        vus: 100,
        duration: '1m',
      },
    },
  };

const file = openSync("./data.csv");

export default async function () {
    const data = await file.readAll();
    console.log(data.byteLength);
}