import Document, { Html, Head, Main, NextScript, DocumentContext } from 'next/document'

class CustomDocument extends Document {
    static async getInitialProps(ctx: DocumentContext) {
        const initialProps = await Document.getInitialProps(ctx)
        return { ...initialProps }
    }

    render() {
        return (
            <Html>
                <Head>
                    <link rel="icon" type="image/png" href="/favicon.png" />
                </Head>
                <body className='antialiased'>
                <Main />
                <NextScript />
                </body>
            </Html>
        )
    }
}

export default CustomDocument
