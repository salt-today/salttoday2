// Defines a comment returned by the scraper.
export type Comment = {
	id: number;
	articleId: number;
	userId: number;
	name: string;
	time: Date;
	text: string;
	likes: number;
	dislikes: number;
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
