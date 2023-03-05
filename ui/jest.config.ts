export const config = {
	preset: 'ts-jest',
	testEnvironment: 'jsdom',
	transform: {
		'^.+\\.svelte$': ['svelte-jester', { preprocess: true }],
		'^.+\\.ts$': ['ts-jest', { tsconfig: 'tsconfig.json' }]
	},
	moduleFileExtensions: ['js', 'ts', 'svelte'],
	setupFilesAfterEnv: ['@testing-library/jest-dom/extend-expect'],
	moduleNameMapper: {
		'^@/(.*)$': '<rootDir>/src/$1',
		'^axios$': '<rootDir>/__mocks__/axios.ts'
	}
};

export default config;
