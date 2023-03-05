// Defines a comment returned by the scraper.
export type Comment = {
	author: string;
	content: string;
	datePosted: Date;
};

// Filter comments based on the following traits.
export type Filters = {
	// 1, 7, 30, 365 (decemeber / red carpet), undefined = all time
	since?: number;
	author?: string;
	liked?: boolean;
	disliked?: boolean;
	onlyDeleted?: boolean;
	// All cities by default
	city?: string;
};
