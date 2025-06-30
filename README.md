[English version](README_en.md)

# microblogen
Jamstack blog generator with microCMS

## 概要
`microblogen` は [microCMS](https://microcms.io/) の記事データを取得し、
テンプレートに差し込んで静的なブログを生成する Go 製ツールです。

## インストール
Go 1.20 以降がインストールされた環境で次のコマンドを実行してビルドできます。

```bash
go build
```

## 使い方
必要な値を環境変数で設定した上で以下のように実行します。

```bash
./microblogen
```

### 使用可能な環境変数
| 変数名 | 説明 | デフォルト |
| ------ | ---- | ---------- |
| `MICROCMS_API_KEY` | microCMS の API キー (必須) | - |
| `SERVICE_DOMAIN` | microCMS のサービスドメイン (必須) | - |
| `EXPORT_PATH` | 出力ディレクトリ (任意) | `./output` |
| `TEMPLATE_PATH` | テンプレートディレクトリ (任意) | `./template` |
| `PAGE_SHOW_LIMIT` | 1 ページに表示する記事数 (任意) | `10` |
| `TIMEZONE` | 日付のタイムゾーン (任意) | `UTC` |
| `CATEGORY_TAG_NAME` | カテゴリ表示時のラベル (任意) | `Category` |
| `TIME_ARCHIVE_NAME` | アーカイブ表示時のラベル (任意) | `Archive` |

## ライセンス
本リポジトリのコードは [MIT License](LICENSE) の下で提供されています。
