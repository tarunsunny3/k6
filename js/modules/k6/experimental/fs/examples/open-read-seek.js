import { openSync, SeekMode } from 'k6/experimental/fs';

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
    let resultString = ""

    let buffer = new Uint8Array(10);
    let n = await file.read(buffer);
    resultString += ab2str(buffer)

    // Read the same data again
    n = await file.read(buffer);
    resultString += ab2str(buffer)

    // Read the same data again
    n = await file.read(buffer);
    resultString += ab2str(buffer)

    await file.seek(0, SeekMode.Start);

    console.log(`[vu ${__VU}] resultString: ${resultString}`);
}

function ab2str(buf) {
    return String.fromCharCode.apply(null, new Uint16Array(buf));
  }