module.exports = {
  roots: ["<rootDir>"],
  transform: {
    "^.+\\.tsx?$": "ts-jest"
  },
  testRunner: "jest-circus/runner",
  testTimeout: 50000,
  testRegex: "(.*|(\\.|/))\\.test\\.tsx?$",
  coveragePathIgnorePatterns: ["dist", "node_modules"],
  moduleFileExtensions: ["ts", "tsx", "js", "jsx", "json", "node"]
};
