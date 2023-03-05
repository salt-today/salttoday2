export default function parseQueryParams(params: any): string {
	return Object.keys(params)
		.map((k) => `${params[k] === undefined || params[k] === null ? '' : `${k}=${params[k]}`}`)
		.join('&');
}
