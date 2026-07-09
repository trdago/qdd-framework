module.exports.handler = async (event, context) => {

    context.callbackWaitsForEmptyEventLoop = false;

    return { statusCode: 200 , ok :true}


}