@module("vitest")
external test: (string, unit => unit) => unit = "test"

@module("vitest")
external expect: 'a => {..} = "expect"

test("app smoke", () => {
  let _ = expect(true)["toBe"](true)
})
