import { readTextFileSync } from "k6/experimental/fs";

export const options = {
    scenarios: {
        default: {
        executor: 'constant-vus',
        vus: 100,
        duration: '1m',
        },
    },
};

const fileText = readTextFileSync("data.bin");

export default async function () {
	console.log(`File text size: ${fileText.length}`);
}
