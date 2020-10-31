// e2e/runWithSelenium.ts
import webdriver from "selenium-webdriver";
import assert from "assert";

let driver: webdriver.WebDriver;

beforeAll(async () => {
  let capabilities: webdriver.Capabilities;
  require("chromedriver");
  capabilities = webdriver.Capabilities.chrome();
  capabilities.set("chromeOptions", {
    args: [
      "--headless",
      "--no-sandbox",
      "--disable-gpu",
      "--window-size=1980,1200",
      "--disable-dev-shm-usage",
      "--use-fake-ui-for-media-stream",
      "--use-fake-device-for-media-stream"
    ]
  });
  driver = await new webdriver.Builder()
    .withCapabilities(capabilities)
    .build();
});

afterAll(async () => {
  await driver.quit()
});

function delay(ms: number) {
  return new Promise( resolve => setTimeout(resolve, ms) );
}

it("E2E", async () => {
  await driver.get('http://localhost/test.html');
  assert.strictEqual(await driver.getTitle(), "E2E");
  await driver.findElement(webdriver.By.id("start_btn")).click()
  await delay(1200);
  await driver.findElement(webdriver.By.id("stop_btn")).click()
});
