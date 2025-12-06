# Gin Fleamarket API

フリーマーケットアプリケーションのバックエンドAPIサーバーです。Go言語とGinフレームワークを使用し、クリーンアーキテクチャに基づいて設計・実装されています。AWSクラウド環境（Aurora Serverless、Lambda、EC2、ECR、API Gateway、IAM、VPC）へのデプロイ実績があります。

## 📋 目次

- [概要](#概要)
- [アーキテクチャ](#アーキテクチャ)
- [技術スタック](#技術スタック)
- [AWSデプロイ実績](#awsデプロイ実績)
- [プロジェクト構造](#プロジェクト構造)
- [機能](#機能)
- [APIエンドポイント](#apiエンドポイント)
- [認証・認可](#認証認可)
- [Docker](#docker)

## 概要

このアプリケーションは、ユーザーが商品を出品・管理できるフリーマーケットプラットフォームのバックエンドAPIです。以下の主要機能を提供します：

- **ユーザー認証・認可**: JWTベースの認証システム（アクセストークン・リフレッシュトークン）
- **商品管理**: 商品の作成、閲覧、更新、削除
- **ロールベースアクセス制御**: 管理者と一般ユーザーの権限分離
- **トークンブラックリスト**: ログアウト済みトークンの無効化

## アーキテクチャ

### クリーンアーキテクチャの採用

本プロジェクトは、**クリーンアーキテクチャ**の原則に基づいて設計されています。各レイヤーが明確に分離され、依存関係が一方向に流れる構造になっています。


### インターフェースによる依存性逆転

各レイヤー間の結合を緩和するため、**インターフェース**を積極的に活用しています：

- **Controller層**: `IItemController`, `IAuthController` インターフェース
- **Service層**: `IItemService`, `IAuthService` インターフェース
- **Repository層**: `IItemRepository`, `IAuthRepository`, `ITokenRepository` インターフェース


### レイヤーの責務

#### Controllers（`controllers/`）
- HTTPリクエストの受信とレスポンスの返却
- リクエストデータのバリデーション（DTOを使用）
- エラーハンドリングとHTTPステータスコードの設定

#### Services（`services/`）
- ビジネスロジックの実装
- トランザクション管理
- バリデーション（ビジネスルール）
- Repository層へのアクセス

#### Repositories（`repositories/`）
- データベースへのアクセス
- CRUD操作の実装
- データ永続化の詳細を隠蔽

#### Models（`models/`）
- ドメインモデルの定義
- GORMによるデータベーススキーマ定義

#### DTOs（`dto/`）
- データ転送オブジェクト
- APIリクエスト/レスポンスの構造定義

#### Middlewares（`middlewares/`）
- 認証ミドルウェア（JWT検証）
- ロールベースアクセス制御ミドルウェア

## 技術スタック

### コア技術

- **Go 1.25.2**: プログラミング言語
- **Gin v1.11.0**: 高性能なWebフレームワーク
- **GORM v1.31.1**: Go用ORM（Object-Relational Mapping）

### データベース

- **PostgreSQL**: メインデータベース（ユーザー、商品データ）
- **SQLite**: トークンブラックリスト用データベース

### 認証・セキュリティ

- **JWT (golang-jwt/jwt/v5)**: JSON Web Tokenによる認証
- **bcrypt**: パスワードハッシュ化
- **CORS**: クロスオリジンリソース共有のサポート

### インフラストラクチャ

- **Docker**: コンテナ化
- **Docker Compose**: ローカル開発環境の構築

## AWSデプロイ実績

本アプリケーションは、以下のAWSサービスを使用して本番環境にデプロイしました：

### 使用したAWSサービス

#### 1. **Amazon Aurora Serverless**
- **用途**: マネージドPostgreSQLデータベース
- **特徴**: 自動スケーリング、高可用性、サーバーレス運用
- **メリット**: コスト最適化、運用負荷の削減

#### 2. **AWS Lambda**
- **用途**: サーバーレス実行環境
- **実装**: Lambda Web Adapterを使用したGinアプリケーションの実行
- **特徴**: 自動スケーリング、従量課金、コールドスタート対策を実装

#### 3. **Amazon EC2**
- **用途**: コンテナ実行環境（オプション）
- **特徴**: ECS/FargateやEC2上でのコンテナ実行に対応

#### 4. **Amazon ECR (Elastic Container Registry)**
- **用途**: Dockerコンテナイメージのレジストリ
- **特徴**: プライベートリポジトリ、イメージのバージョン管理

#### 5. **Amazon API Gateway**
- **用途**: RESTful APIのエンドポイント
- **特徴**: リクエストルーティング、レート制限、認証統合
- **統合**: Lambda関数との統合

#### 6. **AWS IAM (Identity and Access Management)**
- **用途**: アクセス制御とセキュリティ
- **実装**: 
  - Lambda実行ロールの設定
  - Aurora Serverlessへのアクセス権限管理
  - ECRへのプッシュ/プル権限
  - API Gatewayの認証・認可設定

#### 7. **Amazon VPC (Virtual Private Cloud)**
- **用途**: ネットワーク分離とセキュリティ
- **実装**: 
  - Aurora ServerlessをVPC内に配置
  - Lambda関数をVPC内で実行（必要に応じて）
  - セキュリティグループによるアクセス制御
  - プライベートサブネットの活用


### デプロイの特徴

- **マルチ環境対応**: Lambda環境とEC2環境の両方に対応
- **環境変数による設定**: 本番環境と開発環境の切り替え
- **SSL/TLS対応**: 本番環境では`sslmode=require`で接続
- **コールドスタート対策**: 非同期データベース接続による起動時間の短縮

## プロジェクト構造

```
gin-fleamarket/
├── bootstrap/              # ブートストラップ関連
├── constants/             # 定数定義
│   └── constants.go
├── controllers/           # コントローラー層
│   ├── auth_controller.go
│   └── item_controller.go
├── dto/                  # データ転送オブジェクト
│   ├── auth_dto.go
│   └── item_dto.go
├── infra/                # インフラストラクチャ層
│   ├── db.go            # データベース接続設定
│   └── initializer.go
├── middlewares/          # ミドルウェア
│   ├── auth_middleware.go
│   └── role_middleware.go
├── migrations/           # データベースマイグレーション
│   └── migration.go
├── models/              # ドメインモデル
│   ├── blacklisted_token.go
│   ├── item.go
│   └── user.go
├── repositories/        # リポジトリ層（インターフェース + 実装）
│   ├── auth_repository.go
│   ├── item_repository.go
│   └── token_repository.go
├── services/            # サービス層（インターフェース + 実装）
│   ├── auth_service.go
│   └── item_services.go
├── docker/              # Docker関連ファイル
│   ├── postgres/
│   └── pgadmin/
├── docker-compose.yaml  # Docker Compose設定
├── Dockerfile           # Dockerイメージ定義
├── go.mod              # Go依存関係
├── go.sum              # Go依存関係チェックサム
└── main.go             # アプリケーションエントリーポイント
```

## 機能

### 認証機能

- **ユーザー登録（Signup）**
  - メールアドレスとパスワードによる新規登録
  - パスワードはbcryptでハッシュ化
  - 最初のユーザーは自動的に管理者権限を付与

- **ログイン（Login）**
  - メールアドレスとパスワードによる認証
  - アクセストークン（1時間有効）とリフレッシュトークン（7日間有効）を発行

- **トークンリフレッシュ（Refresh Token）**
  - リフレッシュトークンを使用して新しいトークンペアを取得
  - 古いリフレッシュトークンは自動的にブラックリストに追加

- **ログアウト（Logout）**
  - アクセストークンをブラックリストに追加
  - トークンの有効期限までブラックリストに保持

### 商品管理機能

- **商品一覧取得（GET /items）**
  - 認証不要で全商品を取得

- **商品詳細取得（GET /items/:id）**
  - 認証必須
  - 自分の商品のみ取得可能

- **商品作成（POST /items）**
  - 認証必須
  - 作成者は自動的にログインユーザーに設定

- **商品更新（PUT /items/:id）**
  - 認証必須
  - 自分の商品のみ更新可能

- **商品削除（DELETE /items/:id）**
  - 認証必須
  - 管理者のみ削除可能

### ロールベースアクセス制御

- **管理者（admin）**: 全商品の削除が可能
- **一般ユーザー（user）**: 自分の商品の作成・更新・閲覧が可能

## APIエンドポイント

### 認証エンドポイント

#### POST /auth/signup
ユーザー新規登録

**リクエストボディ:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**レスポンス:**
- `200 OK`: 登録成功
- `400 Bad Request`: バリデーションエラー

#### POST /auth/login
ログイン

**リクエストボディ:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**レスポンス:**
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

#### POST /auth/refresh
トークンリフレッシュ

**リクエストボディ:**
```json
{
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**レスポンス:**
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

#### POST /auth/logout
ログアウト

**ヘッダー:**
```
Authorization: Bearer <accessToken>
```

**レスポンス:**
- `200 OK`: ログアウト成功

### 商品エンドポイント

#### GET /items
商品一覧取得（認証不要）

**レスポンス:**
```json
{
  "data": [
    {
      "ID": 1,
      "CreatedAt": "2024-01-01T00:00:00Z",
      "UpdatedAt": "2024-01-01T00:00:00Z",
      "DeletedAt": null,
      "name": "商品名",
      "price": 1000,
      "description": "商品説明",
      "sold_out": false,
      "user_id": 1
    }
  ]
}
```

#### GET /items/:id
商品詳細取得（認証必須）

**ヘッダー:**
```
Authorization: Bearer <accessToken>
```

**レスポンス:**
```json
{
  "data": {
    "ID": 1,
    "name": "商品名",
    "price": 1000,
    "description": "商品説明",
    "sold_out": false,
    "user_id": 1
  }
}
```

#### POST /items
商品作成（認証必須）

**ヘッダー:**
```
Authorization: Bearer <accessToken>
```

**リクエストボディ:**
```json
{
  "name": "商品名",
  "price": 1000,
  "description": "商品説明"
}
```

**レスポンス:**
```json
{
  "data": {
    "ID": 1,
    "name": "商品名",
    "price": 1000,
    "description": "商品説明",
    "sold_out": false,
    "user_id": 1
  }
}
```

#### PUT /items/:id
商品更新（認証必須、自分の商品のみ）

**ヘッダー:**
```
Authorization: Bearer <accessToken>
```

**リクエストボディ:**
```json
{
  "name": "更新された商品名",
  "price": 2000,
  "description": "更新された説明",
  "sold_out": true
}
```

**レスポンス:**
```json
{
  "data": {
    "ID": 1,
    "name": "更新された商品名",
    "price": 2000,
    "description": "更新された説明",
    "sold_out": true,
    "user_id": 1
  }
}
```

#### DELETE /items/:id
商品削除（認証必須、管理者のみ）

**ヘッダー:**
```
Authorization: Bearer <accessToken>
```

**レスポンス:**
- `200 OK`: 削除成功
- `403 Forbidden`: 権限不足
- `404 Not Found`: 商品が見つからない

## 認証・認可

### JWT認証

本アプリケーションは、JWT（JSON Web Token）を使用した認証システムを実装しています。

#### トークンの種類

1. **アクセストークン**
   - 有効期限: 1時間
   - 用途: APIリクエストの認証
   - ペイロード: ユーザーID、メールアドレス、ロール、タイプ（"access"）

2. **リフレッシュトークン**
   - 有効期限: 7日間
   - 用途: アクセストークンの再発行
   - ペイロード: ユーザーID、メールアドレス、ロール、タイプ（"refresh"）

#### トークンブラックリスト

ログアウトやトークンリフレッシュ時に、使用済みトークンをSQLiteデータベースのブラックリストに追加します。これにより、トークンが無効化されるまで（有効期限まで）再利用を防止します。

#### 認証フロー

1. ユーザーがログイン → アクセストークンとリフレッシュトークンを取得
2. APIリクエスト時に`Authorization: Bearer <accessToken>`ヘッダーを付与
3. `AuthMiddleware`がトークンを検証
4. トークンが有効な場合、ユーザー情報をコンテキストに設定
5. リクエスト処理を継続

#### ロールベースアクセス制御

`RoleBasedAccessControl`ミドルウェアを使用して、エンドポイントごとにアクセス権限を制御します。

- **一般ユーザー（user）**: 自分の商品の作成・更新・閲覧が可能
- **管理者（admin）**: 全商品の削除が可能

**重要な実装ポイント:**
- トークンに含まれるロール情報ではなく、**データベースから取得した最新のロール情報**を使用
- これにより、管理者によるロール変更が即座に反映される

## Docker

### Dockerfile

本プロジェクトは、マルチステージビルドを使用した最適化されたDockerfileを提供しています。

**特徴:**
- マルチステージビルドによるイメージサイズの最適化
- distrolessイメージによるセキュリティ向上
- 非rootユーザーでの実行

### Docker Compose

ローカル開発環境用の`docker-compose.yaml`が提供されています：

- **PostgreSQL**: データベースサーバー
- **pgAdmin**: データベース管理ツール

## セキュリティ機能

- **パスワードハッシュ化**: bcryptを使用した安全なパスワード保存
- **JWT署名検証**: HMAC-SHA256によるトークン署名
- **トークンブラックリスト**: ログアウト済みトークンの無効化
- **ロールベースアクセス制御**: エンドポイントごとの権限管理
- **入力バリデーション**: DTOによるリクエストデータの検証
- **SQLインジェクション対策**: GORMによるパラメータ化クエリ
- **CORS設定**: クロスオリジンリクエストの制御
