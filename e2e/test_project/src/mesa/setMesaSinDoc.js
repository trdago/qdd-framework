const { Client } = require('pg')
const { validator:{validateAll} } = require('indicative') 


module.exports.handler = async (event, context) => {

    context.callbackWaitsForEmptyEventLoop = false;
    const client = new Client() 

    try {
        await validateAll( event.body, 
            { 
                'mesa_id': 'required|number',
                'estado':'required|number',
            }, 
            {
                'required':  ' El campo {{ field }} es obligatorio',
                'number':  'El campo {{ field }} no es un numero'
            }
        )

        console.log('llego hasta abajo de la validacion');
        const data=event.body;
        const valuesMesa =[
            data.estado,
            data.mesa_id
        ]

        await client.connect()
        await client.query('BEGIN');
        const updateMesa =  await client.query(
            ` 
            UPDATE mesa SET estado = $1,
            fecha_digitacion = NOW()
            WHERE id = $2;
            `, valuesMesa)
       
        await client.query('COMMIT');
        await client.end();
        
        return { statusCode: 200 , ok:true }
    }catch(error){
        console.error('Fallo la consulta::', error)
        if(client)
        {
            await client.query('ROLLBACK');
            await client.end()

        }

        return { statusCode: 200 , ok : false }

    }

}