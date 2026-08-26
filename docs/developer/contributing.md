# Contributing & License

Contributions to Project Solaria are welcome! Whether improving solar physics models, adding support for new MPPT charge controllers (Victron, EPEVER), or enhancing the PWA dashboard, here is how you can help.

---

## 🤝 Contribution Workflow

1. **Fork the Repository:** Create a fork at `https://github.com/fkcurrie/solaria`.
2. **Create a Feature Branch:** `git checkout -b feature/my-cool-feature`
3. **Ensure All Tests Pass:**
   ```bash
   go test -v ./...
   ./bin/solaria-e2e-audit
   ```
4. **Adhere to Code Safety & Privacy Invariants:**
   - Never commit raw cloud deployment URLs or private tokens.
   - Respect LiFePO4 cold charging protections ($T \le 0^\circ\text{C}$).
5. **Submit a Pull Request:** Open a PR against the `main` branch.

---

## 📄 License

Solaria is open-source software licensed under the **[MIT License](https://github.com/fkcurrie/solaria/blob/main/LICENSE)**.
