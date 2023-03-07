// Parse the object provided into Query Parameters.
export default function parseQueryParams(params: any): string {
	// TODO: URL SafeEncode this?
	return Object.keys(params)
		.map((k) => `${params[k] === undefined || params[k] === null ? '' : `${k}=${params[k]}`}`)
		.join('&');
}
